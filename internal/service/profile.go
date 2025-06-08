package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"gorm.io/gorm"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token has expired")
)

// ProfileService handles user profile operations
type ProfileService struct {
	db *gorm.DB
}

// Ensure ProfileService implements IProfileService
var _ IProfileService = (*ProfileService)(nil)

// NewProfileService creates a new ProfileService instance
func NewProfileService(db *gorm.DB) *ProfileService {
	return &ProfileService{
		db: db,
	}
}

// GetProfile retrieves a user's profile
func (s *ProfileService) GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error) {
	var profile models.UserProfile
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

// UpdateProfile updates a user's profile
func (s *ProfileService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *types.UpdateProfileRequest) (*models.UserProfile, error) {
	var profile models.UserProfile
	if err := s.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Username != "" {
		profile.Username = req.Username
	}
	if req.Bio != nil {
		profile.Bio = *req.Bio
	}
	if req.ProfilePictureURL != nil {
		profile.ProfilePictureURL = *req.ProfilePictureURL
	}
	if req.PrivacyLevel != nil {
		profile.PrivacyLevel = *req.PrivacyLevel
	}

	if err := s.db.Save(&profile).Error; err != nil {
		return nil, err
	}

	return &profile, nil
}

// Logout handles user logout
func (s *ProfileService) Logout(ctx context.Context, userID uuid.UUID) error {
	// In a real implementation, you might want to invalidate the token
	// For now, we'll just return nil as the client will handle token removal
	return nil
}

// GetUserRecipes retrieves a user's recipes
func (s *ProfileService) GetUserRecipes(ctx context.Context, userID uuid.UUID) ([]*models.Recipe, error) {
	var recipes []models.Recipe
	if err := s.db.Where("user_id = ?", userID).Find(&recipes).Error; err != nil {
		return nil, err
	}

	// Convert to []*models.Recipe
	result := make([]*models.Recipe, len(recipes))
	for i := range recipes {
		result[i] = &recipes[i]
	}
	return result, nil
}

// GetProfileHistory retrieves the change history for a user's profile
func (s *ProfileService) GetProfileHistory(ctx context.Context, userID uuid.UUID) ([]*types.ProfileHistory, error) {
	var history []models.ProfileHistory
	if err := s.db.Where("user_id = ?", userID.String()).Find(&history).Error; err != nil {
		return nil, err
	}

	// Convert to types.ProfileHistory
	result := make([]*types.ProfileHistory, len(history))
	for i, h := range history {
		result[i] = &types.ProfileHistory{
			ID:        uuid.MustParse(h.UserID), // Convert string to UUID
			UserID:    uuid.MustParse(h.UserID),
			Field:     h.Field,
			OldValue:  h.OldValue,
			NewValue:  h.NewValue,
			ChangedAt: h.ChangedAt,
			ChangedBy: h.ChangedBy,
		}
	}
	return result, nil
}

// SanitizeProfile sanitizes profile data before sending to client
func (s *ProfileService) SanitizeProfile(profile map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	// Only include safe fields
	if username, ok := profile["username"]; ok {
		sanitized["username"] = username
	}
	if email, ok := profile["email"]; ok {
		sanitized["email"] = email
	}
	if id, ok := profile["id"]; ok {
		sanitized["id"] = id
	}

	return sanitized
}

// RecordProfileChange records a change to a user's profile
func (s *ProfileService) RecordProfileChange(userID, field, oldValue, newValue, changedBy string) error {
	history := &models.ProfileHistory{
		UserID:    userID,
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
		ChangedAt: time.Now(),
		ChangedBy: changedBy,
	}

	return s.db.Create(history).Error
}

// GetUserProfile retrieves a user's profile by username
func (s *ProfileService) GetUserProfile(username string) (*models.UserProfile, error) {
	var profile models.UserProfile
	if err := s.db.Joins("JOIN users ON users.id = user_profiles.user_id").
		Where("users.username = ?", username).
		First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}
