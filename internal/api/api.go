package api

import (
	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"gorm.io/gorm"
)

func SetupAPI(router *gin.Engine, db *gorm.DB, jwtSecret string) {
	v1 := router.Group("/api/v1")
	{
		// Initialize services
		authService := service.NewAuthService(db, jwtSecret)
		emailService := service.NewEmailService()
		profileService := service.NewProfileService(db)
		recipeService := service.NewRecipeService(db, nil)

		// Initialize handlers
		authHandler := NewAuthHandler(authService, emailService, db)
		profileHandler := NewProfileHandler(profileService, authService)
		recipeHandler := NewRecipeHandler(recipeService, authService, nil, nil, db)

		// Register routes
		authHandler.RegisterRoutes(v1)
		profileHandler.RegisterRoutes(v1)
		recipeHandler.RegisterRoutes(v1)
	}
}
