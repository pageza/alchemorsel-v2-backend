package router

import (
	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/api"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
)

// SetupRouter configures the application routes
func SetupRouter(
	authHandler *api.AuthHandler,
	recipeHandler *api.RecipeHandler,
	llmHandler *api.LLMHandler,
) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(middleware.CORS())

	// Public routes
	router.POST("/api/auth/login", authHandler.Login)
	router.POST("/api/auth/register", authHandler.Register)

	// Protected routes
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(authHandler.GetAuthService()))
	{
		// Recipe routes
		protected.GET("/recipes", recipeHandler.ListRecipes)
		protected.GET("/recipes/:id", recipeHandler.GetRecipe)
		protected.POST("/recipes", recipeHandler.CreateRecipe)
		protected.PUT("/recipes/:id", recipeHandler.UpdateRecipe)
		protected.DELETE("/recipes/:id", recipeHandler.DeleteRecipe)

		// LLM routes
		protected.POST("/llm/query", llmHandler.Query)
		protected.GET("/llm/drafts/:id", llmHandler.GetDraft)
		protected.DELETE("/llm/drafts/:id", llmHandler.DeleteDraft)
	}

	return router
}
