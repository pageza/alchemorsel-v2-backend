package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"gorm.io/gorm"
)

// Context key type to avoid collisions
type contextKey string

const usernameContextKey contextKey = "username"

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService  service.IAuthService
	emailService service.IEmailService
	db           *gorm.DB
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService service.IAuthService, emailService service.IEmailService, db *gorm.DB) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		emailService: emailService,
		db:           db,
	}
}

// RegisterRoutes registers the auth routes
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/verify-email", h.VerifyEmail)
		auth.POST("/resend-verification", h.ResendVerification)
		auth.GET("/profile", h.GetProfile)
		auth.PUT("/profile", h.UpdateProfile)
	}
}

// RegisterRequest represents the complete registration request
type RegisterRequest struct {
	Username            string   `json:"username" binding:"required"`
	Email               string   `json:"email" binding:"required,email"`
	Password            string   `json:"password" binding:"required"`
	Name                string   `json:"name"`
	DietaryLifestyles   []string `json:"dietary_lifestyles"`
	CuisinePreferences  []string `json:"cuisine_preferences"`
	Allergies           []string `json:"allergies"`
	DietaryPreferences  []string `json:"dietary_preferences"` // Legacy field support
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prepare user preferences from registration request
	var userPrefs *types.UserPreferences
	if len(req.DietaryLifestyles) > 0 || len(req.DietaryPreferences) > 0 || len(req.Allergies) > 0 {
		// Combine all dietary preferences into one list
		var allDietary []string
		allDietary = append(allDietary, req.DietaryLifestyles...)
		allDietary = append(allDietary, req.DietaryPreferences...)
		
		userPrefs = &types.UserPreferences{
			DietaryPrefs:    allDietary,
			Allergies:       req.Allergies,
			FavoriteCuisine: "", // Will be set from cuisine preferences if any
		}
		
		// Set favorite cuisine from first cuisine preference if provided
		if len(req.CuisinePreferences) > 0 {
			userPrefs.FavoriteCuisine = req.CuisinePreferences[0]
		}
	}

	// Create context with username
	ctx := context.WithValue(c.Request.Context(), usernameContextKey, req.Username)
	user, err := h.authService.Register(ctx, req.Email, req.Password, req.Username, userPrefs)
	if err != nil {
		// Handle specific error cases with appropriate HTTP status codes
		switch err.Error() {
		case "user already exists":
			c.JSON(http.StatusConflict, gin.H{"error": "An account with this email already exists"})
		case "username already taken":
			c.JSON(http.StatusConflict, gin.H{"error": "This username is already taken"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed. Please try again later."})
		}
		return
	}

	// Generate verification token and send verification email
	verificationToken, err := h.authService.GenerateVerificationToken(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate verification token"})
		return
	}

	// Send verification email
	if err := h.emailService.SendVerificationEmail(user, verificationToken); err != nil {
		// Log error but don't fail registration - user can request resend later
		c.Header("X-Warning", "User created but verification email failed to send")
	}

	claims := &types.TokenClaims{
		UserID:          user.ID,
		Username:        req.Username,
		IsEmailVerified: user.EmailVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(user.CreatedAt.Add(3 * 3600 * 1e9)), // 3 hours for improved security
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
		"message": "Registration successful. Please check your email to verify your account.",
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
		// Handle specific error cases with appropriate messages
		switch err.Error() {
		case "invalid credentials":
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed. Please try again later."})
		}
		return
	}

	claims := &types.TokenClaims{
		UserID:          user.ID,
		Username:        profile.Username,
		IsEmailVerified: user.EmailVerified,
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

// VerifyEmail handles email verification
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.ValidateVerificationToken(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Send welcome email after successful verification
	if err := h.emailService.SendWelcomeEmail(user); err != nil {
		// Log error but don't fail the verification
		// User is still verified even if welcome email fails
		c.Header("X-Warning", "Email verified but welcome email failed to send")
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Email verified successfully",
		"user_id":     user.ID,
		"email":       user.Email,
		"verified_at": user.EmailVerifiedAt,
	})
}

// ResendVerification handles resending verification email
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.ResendVerificationEmail(c.Request.Context(), req.Email, h.emailService)
	if err != nil {
		// Don't reveal whether user exists or not for security
		if err.Error() == "user not found" {
			c.JSON(http.StatusOK, gin.H{
				"message": "If the email exists in our system, a verification email has been sent",
			})
			return
		}
		if err.Error() == "email already verified" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email is already verified"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification email sent successfully",
	})
}
