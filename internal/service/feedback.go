package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"gorm.io/gorm"
)

type FeedbackService struct {
	db           *gorm.DB
	emailService IEmailService
}

func NewFeedbackService(db *gorm.DB, emailService IEmailService) IFeedbackService {
	return &FeedbackService{
		db:           db,
		emailService: emailService,
	}
}

func (s *FeedbackService) CreateFeedback(ctx context.Context, req *types.CreateFeedbackRequest, userID *uuid.UUID) (*models.Feedback, error) {
	feedback := &models.Feedback{
		UserID:      userID,
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		UserAgent:   req.UserAgent,
		URL:         req.URL,
		Status:      "open",
	}

	// Set default priority if not provided
	if feedback.Priority == "" {
		feedback.Priority = "medium"
	}

	// Create feedback record
	if err := s.db.WithContext(ctx).Create(feedback).Error; err != nil {
		return nil, fmt.Errorf("failed to create feedback: %w", err)
	}

	// Load user information for email notification
	var user *models.User
	if userID != nil {
		if err := s.db.WithContext(ctx).First(&user, "id = ?", *userID).Error; err != nil {
			// Log error but don't fail the feedback creation
			fmt.Printf("Warning: Could not load user for feedback notification: %v\n", err)
		}
	}

	// Send email notification asynchronously
	go func() {
		if err := s.emailService.SendFeedbackNotification(feedback, user); err != nil {
			fmt.Printf("Error sending feedback notification: %v\n", err)
		}
	}()

	return feedback, nil
}

func (s *FeedbackService) GetFeedback(ctx context.Context, id uuid.UUID) (*models.Feedback, error) {
	var feedback models.Feedback
	if err := s.db.WithContext(ctx).Preload("User").First(&feedback, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("feedback not found")
		}
		return nil, fmt.Errorf("failed to get feedback: %w", err)
	}
	return &feedback, nil
}

func (s *FeedbackService) ListFeedback(ctx context.Context, filters *models.FeedbackFilters) ([]*models.Feedback, error) {
	query := s.db.WithContext(ctx).Preload("User")

	// Apply filters
	if filters != nil {
		if filters.Type != "" {
			query = query.Where("type = ?", filters.Type)
		}
		if filters.Status != "" {
			query = query.Where("status = ?", filters.Status)
		}
		if filters.Priority != "" {
			query = query.Where("priority = ?", filters.Priority)
		}
		if filters.UserID != "" {
			if userUUID, err := uuid.Parse(filters.UserID); err == nil {
				query = query.Where("user_id = ?", userUUID)
			}
		}

		// Apply pagination
		if filters.Limit > 0 {
			query = query.Limit(filters.Limit)
		} else {
			query = query.Limit(50) // Default limit
		}
		if filters.Offset > 0 {
			query = query.Offset(filters.Offset)
		}
	}

	// Order by creation date (newest first)
	query = query.Order("created_at DESC")

	var feedback []*models.Feedback
	if err := query.Find(&feedback).Error; err != nil {
		return nil, fmt.Errorf("failed to list feedback: %w", err)
	}

	return feedback, nil
}

func (s *FeedbackService) UpdateFeedbackStatus(ctx context.Context, id uuid.UUID, status string, adminNotes string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if adminNotes != "" {
		updates["admin_notes"] = adminNotes
	}

	result := s.db.WithContext(ctx).Model(&models.Feedback{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update feedback status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("feedback not found")
	}

	return nil
}
