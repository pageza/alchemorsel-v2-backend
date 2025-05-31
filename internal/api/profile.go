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
		profile.GET("/dietary-preferences", h.GetDietaryPreferences)
		profile.PUT("/dietary-preferences", h.UpdateDietaryPreferences)
		profile.GET("/allergens", h.GetAllergens)
		profile.PUT("/allergens", h.UpdateAllergens)
		profile.GET("/appliances", h.GetAppliances)
		profile.PUT("/appliances", h.UpdateAppliances)
	}
}

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	profile, err := h.profileService.GetProfile(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.profileService.UpdateProfile(userID.(uuid.UUID), updates); err != nil {
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

func (h *ProfileHandler) GetDietaryPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	preferences, err := h.profileService.GetDietaryPreferences(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get dietary preferences"})
		return
	}

	c.JSON(http.StatusOK, preferences)
}

func (h *ProfileHandler) UpdateDietaryPreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var preferences []models.DietaryPreference
	if err := c.ShouldBindJSON(&preferences); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.profileService.UpdateDietaryPreferences(userID.(uuid.UUID), preferences); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update dietary preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dietary preferences updated successfully"})
}

func (h *ProfileHandler) GetAllergens(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	allergens, err := h.profileService.GetAllergens(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get allergens"})
		return
	}

	c.JSON(http.StatusOK, allergens)
}

func (h *ProfileHandler) UpdateAllergens(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var allergens []models.Allergen
	if err := c.ShouldBindJSON(&allergens); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.profileService.UpdateAllergens(userID.(uuid.UUID), allergens); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update allergens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "allergens updated successfully"})
}

func (h *ProfileHandler) GetAppliances(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	appliances, err := h.profileService.GetAppliances(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get appliances"})
		return
	}

	c.JSON(http.StatusOK, appliances)
}

func (h *ProfileHandler) UpdateAppliances(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var appliances []models.UserAppliance
	if err := c.ShouldBindJSON(&appliances); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.profileService.UpdateAppliances(userID.(uuid.UUID), appliances); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update appliances"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "appliances updated successfully"})
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
