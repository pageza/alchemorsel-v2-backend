package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
)

// Profile represents a user profile
type Profile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type ProfileHandler struct {
	profileService service.IProfileService
	authService    service.IAuthService
}

func NewProfileHandler(profileService service.IProfileService, authService service.IAuthService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
		authService:    authService,
	}
}

func (h *ProfileHandler) RegisterRoutes(router *gin.RouterGroup) {
	profile := router.Group("/profile")
	profile.Use(middleware.AuthMiddleware(h.authService))
	{
		profile.GET("", h.GetProfile)
		profile.PUT("", h.UpdateProfile)
		profile.POST("/logout", h.Logout)
		profile.GET("/recipes", h.GetUserRecipes)
		profile.GET("/history", h.GetProfileHistory)
	}
}

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	// Get user with email verification status
	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.profileService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	recipes, err := h.profileService.GetUserRecipes(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Combine user and profile data
	profileData := gin.H{
		"id":                  user.ID,
		"name":                user.Name,
		"email":               user.Email,
		"email_verified":      user.EmailVerified,
		"email_verified_at":   user.EmailVerifiedAt,
		"username":            profile.Username,
		"bio":                 profile.Bio,
		"profile_picture_url": profile.ProfilePictureURL,
		"privacy_level":       profile.PrivacyLevel,
		"created_at":          user.CreatedAt,
		"updated_at":          user.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"profile": profileData,
		"recipes": recipes,
	})
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	var req types.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	profile, err := h.profileService.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) Logout(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.profileService.Logout(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *ProfileHandler) GetUserRecipes(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	recipes, err := h.profileService.GetUserRecipes(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recipes)
}

func (h *ProfileHandler) GetProfileHistory(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	history, err := h.profileService.GetProfileHistory(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, history)
}

// RegisterProfileRoutes registers the profile API routes
func RegisterProfileRoutes(router *gin.Engine, profileService service.IProfileService, authService service.IAuthService) {
	handler := NewProfileHandler(profileService, authService)

	profile := router.Group("/api/v1/profile")
	profile.Use(middleware.AuthMiddleware(authService))
	{
		profile.GET("", handler.GetProfile)
		profile.PUT("", handler.UpdateProfile)
	}
}
