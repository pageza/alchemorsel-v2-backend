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
		profileService := service.NewProfileService(db, jwtSecret)

		// Initialize handlers
		authHandler := NewAuthHandler(authService)
		profileHandler := NewProfileHandler(profileService)

		// Register routes
		authHandler.RegisterRoutes(v1)
		profileHandler.RegisterRoutes(v1)
	}
}
