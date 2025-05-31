package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	GetProfile(userID uint) (*models.Profile, error)
	UpdateProfile(userID uint, profile *models.Profile) error
	GetProfileHistory(userID uint) ([]map[string]interface{}, error)
	ValidateToken(token string) (*middleware.TokenClaims, error)
}

type ProfileHandler struct {
	profileService ProfileService
}

func NewProfileHandler(profileService ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	profile, err := h.profileService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	var profile models.Profile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	profile.UserID = userID
	if err := h.profileService.UpdateProfile(userID, &profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profile updated successfully"})
}

func (h *ProfileHandler) GetProfileHistory(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	history, err := h.profileService.GetProfileHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// RegisterProfileRoutes registers the profile API routes
func RegisterProfileRoutes(router *gin.Engine, profileService ProfileService) {
	handler := NewProfileHandler(profileService)

	profile := router.Group("/api/v1/profile")
	profile.Use(middleware.AuthMiddleware(profileService))
	{
		profile.GET("", handler.GetProfile)
		profile.PUT("", handler.UpdateProfile)
		profile.GET("/history", handler.GetProfileHistory)
	}
}
