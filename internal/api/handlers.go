package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"gorm.io/gorm"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(router *gin.Engine, db *gorm.DB, authService service.IAuthService, llmService service.LLMServiceInterface, embeddingService service.EmbeddingServiceInterface) {
	// Create handlers
	authHandler := NewAuthHandler(authService, db)
	recipeHandler := NewRecipeHandler(service.NewRecipeService(db, embeddingService), authService, llmService, embeddingService)

	// Register routes
	v1 := router.Group("/api/v1")
	authHandler.RegisterRoutes(v1)
	recipeHandler.RegisterRoutes(v1)
}
