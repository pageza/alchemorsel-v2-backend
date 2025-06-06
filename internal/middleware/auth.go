package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TokenValidator interface {
	ValidateToken(token string) (*TokenClaims, error)
}

type TokenClaims struct {
	UserID   uuid.UUID
	Username string
}

// AuthMiddleware creates a middleware that validates JWT tokens
func AuthMiddleware(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		fmt.Printf("[AuthMiddleware] Authorization header: %s\n", authHeader)
		if authHeader == "" {
			fmt.Println("[AuthMiddleware] Missing Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Extract the token from the Authorization header
		// Format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			fmt.Printf("[AuthMiddleware] Invalid Authorization header format: %v\n", parts)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		fmt.Printf("[AuthMiddleware] Extracted token: %s\n", tokenString)
		claims, err := validator.ValidateToken(tokenString)
		if err != nil {
			fmt.Printf("[AuthMiddleware] Token validation error: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		fmt.Printf("[AuthMiddleware] Token valid. UserID: %v, Username: %s\n", claims.UserID, claims.Username)
		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
