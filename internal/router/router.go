package router

import (
	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/api"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

// SetupRouter configures the application routes
func SetupRouter(
	authHandler *api.AuthHandler,
	recipeHandler *api.RecipeHandler,
	llmHandler *api.LLMHandler,
	authService service.IAuthService,
) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(middleware.CORS())

	// API v1 routes
	v1 := router.Group("/api/v1")

	// Auth routes
	auth := v1.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
	}

	// Protected routes
	protected := v1.Group("")
	protected.Use(middleware.AuthMiddleware(authService))
	{
		// Profile routes
		profile := protected.Group("/profile")
		{
			profile.GET("", authHandler.GetProfile)
			profile.PUT("", authHandler.UpdateProfile)
		}

		// Recipe routes
		recipes := protected.Group("/recipes")
		{
			recipes.GET("", recipeHandler.ListRecipes)
			recipes.GET("/:id", recipeHandler.GetRecipe)
			recipes.POST("", recipeHandler.CreateRecipe)
			recipes.PUT("/:id", recipeHandler.UpdateRecipe)
			recipes.DELETE("/:id", recipeHandler.DeleteRecipe)
		}

		// LLM routes
		llm := protected.Group("/llm")
		{
			llm.POST("/query", llmHandler.Query)
			llm.GET("/drafts/:id", llmHandler.GetDraft)
			llm.DELETE("/drafts/:id", llmHandler.DeleteDraft)
		}
	}

	return router
}
