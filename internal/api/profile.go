package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
)

// Profile represents a user profile
type Profile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type ProfileService interface {
	GetUserProfile(userID uuid.UUID) (*models.UserProfile, error)
	UpdateUserProfile(userID uuid.UUID, profile *models.UserProfile) error
	Logout(userID uuid.UUID) error
	ValidateToken(token string) (*middleware.TokenClaims, error)
	GetUserRecipes(userID uuid.UUID) ([]models.Recipe, error)
}

type ProfileHandler struct {
	profileService ProfileService
}

func NewProfileHandler(profileService ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

func (h *ProfileHandler) RegisterRoutes(router *gin.RouterGroup) {
	profile := router.Group("/profile")
	profile.Use(middleware.AuthMiddleware(h.profileService))
	{
		profile.GET("", h.GetProfile)
		profile.PUT("", h.UpdateProfile)
		profile.POST("/logout", h.Logout)
	}
}

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	profile, err := h.profileService.GetUserProfile(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile"})
		return
	}

	recipes, err := h.profileService.GetUserRecipes(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recipes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"profile": profile,
		"recipes": recipes,
	})
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var profile models.UserProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.profileService.UpdateUserProfile(userID.(uuid.UUID), &profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated successfully"})
}

func (h *ProfileHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.profileService.Logout(userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

// RegisterProfileRoutes registers the profile API routes
func RegisterProfileRoutes(router *gin.Engine, profileService ProfileService) {
	handler := NewProfileHandler(profileService)

	profile := router.Group("/api/v1/profile")
	profile.Use(middleware.AuthMiddleware(profileService))
	{
		profile.GET("", handler.GetProfile)
		profile.PUT("", handler.UpdateProfile)
	}
}
