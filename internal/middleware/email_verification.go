package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"gorm.io/gorm"
)

// RequireEmailVerification creates a middleware that requires email verification
func RequireEmailVerification(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by auth middleware)
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Get user from database to check verification status
		var user models.User
		if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify user status"})
			c.Abort()
			return
		}

		// Check if email is verified
		if !user.EmailVerified {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "email verification required",
				"message": "Please verify your email address to access this feature",
				"email":   user.Email,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalEmailVerification creates a middleware that adds verification status to context
// but doesn't block unverified users (useful for features that work differently for verified users)
func OptionalEmailVerification(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by auth middleware)
		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			c.Next()
			return
		}

		// Get user from database to check verification status
		var user models.User
		if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
			c.Next()
			return
		}

		// Add verification status to context
		c.Set("email_verified", user.EmailVerified)
		c.Set("user_email", user.Email)

		c.Next()
	}
}
