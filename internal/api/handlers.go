package api

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/database"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"gorm.io/gorm"
	"log"
	"net/http"
)

// HealthCheck returns the health status of the API
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Alchemorsel API is running",
		"version": "v1.0.0",
	})
}

// RegisterRoutes registers all API routes
func RegisterRoutes(router *gin.Engine, db *gorm.DB, authService service.IAuthService, llmService service.LLMServiceInterface, embeddingService service.EmbeddingServiceInterface, cfg *config.Config) {
	// Health check endpoint (no auth required)
	router.GET("/health", HealthCheck)
	router.GET("/api/health", HealthCheck)

	// Initialize Redis for rate limiting
	redisClient, err := database.NewRedisClient(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis for rate limiting: %v", err)
		// Continue without rate limiting if Redis is not available
		redisClient = nil
	}

	// Create rate limiters
	var recipeCreationLimiter *middleware.RateLimiter
	var recipeModificationLimiter *middleware.RateLimiter

	if redisClient != nil {
		recipeCreationLimiter = middleware.NewRecipeCreationRateLimiter(redisClient)
		recipeModificationLimiter = middleware.NewRecipeModificationRateLimiter(redisClient)
	}

	// Create email and feedback services
	emailService := service.NewEmailService()
	feedbackService := service.NewFeedbackService(db, emailService)

	// Create S3 config and image service
	s3Config, err := config.NewS3Config(context.Background())
	if err != nil {
		log.Printf("Warning: Failed to initialize S3 config: %v", err)
		s3Config = nil
	}
	
	var imageService service.IImageService
	if s3Config != nil {
		imageService, err = service.NewImageService(s3Config)
		if err != nil {
			log.Printf("Warning: Failed to create image service: %v", err)
			imageService = nil
		}
	}

	// Create handlers
	authHandler := NewAuthHandler(authService, emailService, db)
	recipeHandler := NewRecipeHandlerWithRateLimit(service.NewRecipeService(db, embeddingService), authService, llmService, embeddingService, db, recipeCreationLimiter, recipeModificationLimiter)
	// Update LLM service with image service if available
	if imageService != nil {
		llmServiceWithImage, err := service.NewLLMServiceWithServices(embeddingService, imageService)
		if err != nil {
			log.Printf("Warning: Failed to create LLM service with image service: %v", err)
		} else {
			llmService = llmServiceWithImage
		}
	}
	
	llmHandler := NewLLMHandlerWithRateLimit(db, authService.(*service.AuthService), llmService, service.NewRecipeService(db, embeddingService), recipeCreationLimiter)
	profileHandler := NewProfileHandler(service.NewProfileService(db), authService)
	dashboardHandler := NewDashboardHandler(db, authService)
	feedbackHandler := NewFeedbackHandler(feedbackService, db)
	
	// Create image handler if image service is available
	var imageHandler *ImageHandler
	if imageService != nil {
		imageHandler = NewImageHandler(db, imageService, llmService, authService.(*service.AuthService), recipeCreationLimiter)
	}

	fmt.Println("DEBUG: Feedback handler created successfully")

	// Register routes
	v1 := router.Group("/api/v1")
	authHandler.RegisterRoutes(v1)
	recipeHandler.RegisterRoutes(v1)
	llmHandler.RegisterRoutes(v1)
	profileHandler.RegisterRoutes(v1)
	
	// Register image routes if image handler is available
	if imageHandler != nil {
		imageHandler.RegisterRoutes(v1)
	}

	// Feedback routes (supports both authenticated and anonymous)
	fmt.Println("DEBUG: Registering feedback routes")
	feedbackHandler.RegisterRoutes(v1)
	fmt.Println("DEBUG: Feedback routes registered")

	// Dashboard routes (with auth middleware)
	dashboardGroup := v1.Group("")
	dashboardGroup.Use(middleware.AuthMiddleware(authService))
	dashboardHandler.RegisterRoutes(dashboardGroup)

	// Rate limit status endpoint
	if recipeCreationLimiter != nil {
		RegisterRateLimitRoutes(v1, authService, recipeCreationLimiter, recipeModificationLimiter)
	}
}

// RegisterRateLimitRoutes registers endpoints for checking rate limit status
func RegisterRateLimitRoutes(router *gin.RouterGroup, authService service.IAuthService, creationLimiter *middleware.RateLimiter, modificationLimiter *middleware.RateLimiter) {
	rateLimits := router.Group("/rate-limits")
	rateLimits.Use(middleware.AuthMiddleware(authService))
	{
		rateLimits.GET("/recipe-creation", func(c *gin.Context) {
			userID, exists := c.Get("user_id")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
				return
			}

			userIDStr := fmt.Sprintf("%v", userID)
			remaining, resetTime, err := creationLimiter.GetRemainingRequests(c.Request.Context(), userIDStr)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check rate limit"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"limit":      2,
				"remaining":  remaining,
				"reset_time": resetTime.Unix(),
				"window":     "1h",
			})
		})

		rateLimits.GET("/recipe-modification/:recipe_id", func(c *gin.Context) {
			userID, exists := c.Get("user_id")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
				return
			}

			recipeID := c.Param("recipe_id")
			if recipeID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "recipe ID is required"})
				return
			}

			userIDStr := fmt.Sprintf("%v", userID)
			key := fmt.Sprintf("%s:%s", userIDStr, recipeID)

			remaining, resetTime, err := modificationLimiter.GetRemainingRequests(c.Request.Context(), key)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check rate limit"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"limit":      10,
				"remaining":  remaining,
				"reset_time": resetTime.Unix(),
				"window":     "1h",
				"recipe_id":  recipeID,
			})
		})
	}
}
