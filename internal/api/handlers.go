package api

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"gorm.io/gorm"
)

// HealthCheck returns the health status of the API
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"message": "Alchemorsel API is running",
		"version": "v1.0.0",
	})
}

// RegisterRoutes registers all API routes
func RegisterRoutes(router *gin.Engine, db *gorm.DB, authService service.IAuthService, llmService service.LLMServiceInterface, embeddingService service.EmbeddingServiceInterface) {
	// Health check endpoint (no auth required)
	router.GET("/health", HealthCheck)
	router.GET("/api/health", HealthCheck)

	// Create handlers
	authHandler := NewAuthHandler(authService, db)
	recipeHandler := NewRecipeHandler(service.NewRecipeService(db, embeddingService), authService, llmService, embeddingService)
	llmHandler := NewLLMHandler(db, authService.(*service.AuthService), llmService)
	profileHandler := NewProfileHandler(service.NewProfileService(db), authService)

	// Register routes
	v1 := router.Group("/api/v1")
	authHandler.RegisterRoutes(v1)
	recipeHandler.RegisterRoutes(v1)
	llmHandler.RegisterRoutes(v1)
	profileHandler.RegisterRoutes(v1)
}
