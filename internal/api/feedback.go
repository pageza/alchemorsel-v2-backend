package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"gorm.io/gorm"
)

type FeedbackHandler struct {
	feedbackService service.IFeedbackService
	db              *gorm.DB
}

func NewFeedbackHandler(feedbackService service.IFeedbackService, db *gorm.DB) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackService: feedbackService,
		db:              db,
	}
}

func (h *FeedbackHandler) RegisterRoutes(router *gin.RouterGroup) {
	fmt.Println("DEBUG: Inside FeedbackHandler.RegisterRoutes")
	feedback := router.Group("/feedback")
	{
		feedback.POST("/", h.CreateFeedback)              // Authenticated
		feedback.GET("/", h.ListFeedback)                 // Admin only
		feedback.GET("/:id", h.GetFeedback)               // Admin only
		feedback.PUT("/:id/status", h.UpdateStatus)       // Admin only
	}
	fmt.Println("DEBUG: Feedback routes registered successfully")
}

// CreateFeedback creates a new feedback submission
func (h *FeedbackHandler) CreateFeedback(c *gin.Context) {
	var req types.CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (if authenticated)
	var userID *uuid.UUID
	if userIDValue, exists := c.Get("userID"); exists {
		if uid, ok := userIDValue.(uuid.UUID); ok {
			userID = &uid
		}
	}

	// Allow anonymous feedback - userID can be nil

	// Create feedback
	feedback, err := h.feedbackService.CreateFeedback(c.Request.Context(), &req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create feedback"})
		return
	}

	// Convert to response
	response := h.feedbackToResponse(feedback)
	c.JSON(http.StatusCreated, response)
}

// ListFeedback lists all feedback (admin only)
func (h *FeedbackHandler) ListFeedback(c *gin.Context) {
	// Check admin role
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Parse query parameters
	filters := &models.FeedbackFilters{}
	if typeParam := c.Query("type"); typeParam != "" {
		filters.Type = typeParam
	}
	if statusParam := c.Query("status"); statusParam != "" {
		filters.Status = statusParam
	}
	if priorityParam := c.Query("priority"); priorityParam != "" {
		filters.Priority = priorityParam
	}
	if userIDParam := c.Query("user_id"); userIDParam != "" {
		filters.UserID = userIDParam
	}
	if limitParam := c.Query("limit"); limitParam != "" {
		if limit, err := strconv.Atoi(limitParam); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if offset, err := strconv.Atoi(offsetParam); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Get feedback list
	feedbackList, err := h.feedbackService.ListFeedback(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list feedback"})
		return
	}

	// Convert to response
	responses := make([]types.FeedbackResponse, len(feedbackList))
	for i, feedback := range feedbackList {
		responses[i] = h.feedbackToResponse(feedback)
	}

	c.JSON(http.StatusOK, responses)
}

// GetFeedback gets a specific feedback item (admin only)
func (h *FeedbackHandler) GetFeedback(c *gin.Context) {
	// Check admin role
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Parse feedback ID
	idParam := c.Param("id")
	feedbackID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feedback ID"})
		return
	}

	// Get feedback
	feedback, err := h.feedbackService.GetFeedback(c.Request.Context(), feedbackID)
	if err != nil {
		if err.Error() == "feedback not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Feedback not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get feedback"})
		return
	}

	// Convert to response
	response := h.feedbackToResponse(feedback)
	c.JSON(http.StatusOK, response)
}

// UpdateStatus updates the status of a feedback item (admin only)
func (h *FeedbackHandler) UpdateStatus(c *gin.Context) {
	// Check admin role
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Parse feedback ID
	idParam := c.Param("id")
	feedbackID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid feedback ID"})
		return
	}

	// Parse request body
	var req types.UpdateFeedbackStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update feedback status
	err = h.feedbackService.UpdateFeedbackStatus(c.Request.Context(), feedbackID, req.Status, req.AdminNotes)
	if err != nil {
		if err.Error() == "feedback not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Feedback not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update feedback status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Feedback status updated successfully"})
}

// Helper function to check if user is admin
func (h *FeedbackHandler) isAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	if !exists {
		return false
	}
	return role == "admin"
}

// Helper function to convert Feedback model to response
func (h *FeedbackHandler) feedbackToResponse(feedback *models.Feedback) types.FeedbackResponse {
	return types.FeedbackResponse{
		ID:          feedback.ID,
		Type:        feedback.Type,
		Title:       feedback.Title,
		Description: feedback.Description,
		Priority:    feedback.Priority,
		Status:      feedback.Status,
		UserAgent:   feedback.UserAgent,
		URL:         feedback.URL,
		AdminNotes:  feedback.AdminNotes,
		CreatedAt:   feedback.CreatedAt,
		UserID:      feedback.UserID,
	}
}