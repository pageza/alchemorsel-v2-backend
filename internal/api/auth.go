package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"gorm.io/gorm"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService service.IAuthService
	db          *gorm.DB
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService service.IAuthService, db *gorm.DB) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		db:          db,
	}
}

// RegisterRoutes registers the auth routes
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.GET("/profile", h.GetProfile)
		auth.PUT("/profile", h.UpdateProfile)
	}
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Email, req.Password, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	claims := &types.TokenClaims{
		UserID:   user.ID,
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(user.CreatedAt.Add(24 * 3600 * 1e9)),
			IssuedAt:  jwt.NewNumericDate(user.CreatedAt),
		},
	}
	token, err := h.authService.GenerateToken(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id": user.ID,
		"token":   token,
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, profile, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	claims := &types.TokenClaims{
		UserID:   user.ID,
		Username: profile.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(user.CreatedAt.Add(24 * 3600 * 1e9)),
			IssuedAt:  jwt.NewNumericDate(user.CreatedAt),
		},
	}
	token, err := h.authService.GenerateToken(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
		"token":   token,
	})
}

// GetProfile handles getting user profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get user profile from profile service
	var profile models.UserProfile
	if err := h.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  profile.ID,
		"user_id":             profile.UserID,
		"username":            profile.Username,
		"bio":                 profile.Bio,
		"profile_picture_url": profile.ProfilePictureURL,
		"privacy_level":       profile.PrivacyLevel,
	})
}

// UpdateProfile handles updating user profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req types.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update user profile
	var profile models.UserProfile
	if err := h.db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user profile"})
		return
	}

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

	if err := h.db.Save(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  profile.ID,
		"user_id":             profile.UserID,
		"username":            profile.Username,
		"bio":                 profile.Bio,
		"profile_picture_url": profile.ProfilePictureURL,
		"privacy_level":       profile.PrivacyLevel,
	})
}
