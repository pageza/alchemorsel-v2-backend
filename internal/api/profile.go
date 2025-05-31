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
	GetProfile(userID uuid.UUID) (*models.UserProfile, error)
	UpdateProfile(userID uuid.UUID, updates map[string]interface{}) error
	GetDietaryPreferences(userID uuid.UUID) ([]models.DietaryPreference, error)
	UpdateDietaryPreferences(userID uuid.UUID, preferences []models.DietaryPreference) error
	GetAllergens(userID uuid.UUID) ([]models.Allergen, error)
	UpdateAllergens(userID uuid.UUID, allergens []models.Allergen) error
	GetAppliances(userID uuid.UUID) ([]models.UserAppliance, error)
	UpdateAppliances(userID uuid.UUID, appliances []models.UserAppliance) error
	Logout(userID uuid.UUID) error
	ValidateToken(token string) (*middleware.TokenClaims, error)
}

type ProfileHandler struct {
	profileService ProfileService
	authService    middleware.TokenValidator
}

func NewProfileHandler(profileService ProfileService, authService middleware.TokenValidator) *ProfileHandler {
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
		profile.GET("/dietary-preferences", h.GetDietaryPreferences)
		profile.PUT("/dietary-preferences", h.UpdateDietaryPreferences)
		profile.GET("/allergens", h.GetAllergens)
		profile.PUT("/allergens", h.UpdateAllergens)
		profile.GET("/appliances", h.GetAppliances)
		profile.PUT("/appliances", h.UpdateAppliances)
	}
}

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
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
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.profileService.UpdateProfile(userID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated successfully"})
}

func (h *ProfileHandler) Logout(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	if err := h.profileService.Logout(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

func (h *ProfileHandler) GetDietaryPreferences(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	preferences, err := h.profileService.GetDietaryPreferences(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get dietary preferences"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

func (h *ProfileHandler) UpdateDietaryPreferences(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var preferences []models.DietaryPreference
	if err := c.ShouldBindJSON(&preferences); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.profileService.UpdateDietaryPreferences(userID, preferences); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update dietary preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dietary preferences updated successfully"})
}

func (h *ProfileHandler) GetAllergens(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	allergens, err := h.profileService.GetAllergens(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get allergens"})
		return
	}

	c.JSON(http.StatusOK, allergens)
}

func (h *ProfileHandler) UpdateAllergens(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var allergens []models.Allergen
	if err := c.ShouldBindJSON(&allergens); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.profileService.UpdateAllergens(userID, allergens); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update allergens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "allergens updated successfully"})
}

func (h *ProfileHandler) GetAppliances(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	appliances, err := h.profileService.GetAppliances(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get appliances"})
		return
	}

	c.JSON(http.StatusOK, appliances)
}

func (h *ProfileHandler) UpdateAppliances(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	var appliances []models.UserAppliance
	if err := c.ShouldBindJSON(&appliances); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.profileService.UpdateAppliances(userID, appliances); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update appliances"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "appliances updated successfully"})
}

// RegisterProfileRoutes registers the profile API routes
func RegisterProfileRoutes(router *gin.Engine, profileService ProfileService, authService middleware.TokenValidator) {
	handler := NewProfileHandler(profileService, authService)

	profile := router.Group("/api/v1/profile")
	profile.Use(middleware.AuthMiddleware(authService))
	{
		profile.GET("", handler.GetProfile)
		profile.PUT("", handler.UpdateProfile)
	}
}
