package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/api"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

// Server represents the HTTP server
type Server struct {
	router *gin.Engine
	http   *http.Server
	db     *gorm.DB
	logger *log.Logger
}

// New creates a new Server instance
func New(cfg *config.Config, db *gorm.DB) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Add CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "Cache-Control", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	return &Server{
		router: router,
		db:     db,
		logger: log.New(log.Writer(), "[SERVER] ", log.LstdFlags),
	}
}

// Start starts the HTTP server
func (s *Server) Start(cfg *config.Config) error {
	// Initialize services
	profileService := service.NewProfileService(s.db, cfg.JWTSecret)
	authService := service.NewAuthService(s.db, cfg.JWTSecret)

	// Initialize handlers
	profileHandler := api.NewProfileHandler(profileService)
	authHandler := api.NewAuthHandler(authService)
	recipeHandler, err := api.NewRecipeHandler(s.db, authService)
	if err != nil {
		return fmt.Errorf("failed to create recipe handler: %w", err)
	}

	llmHandler, err := api.NewLLMHandler(s.db, authService)
	if err != nil {
		return fmt.Errorf("failed to create LLM handler: %w", err)
	}

	apiGroup := s.router.Group("/api/v1")
	profileHandler.RegisterRoutes(apiGroup)
	authHandler.RegisterRoutes(apiGroup)
	recipeHandler.RegisterRoutes(apiGroup)
	llmHandler.RegisterRoutes(apiGroup)

	// Add health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Create HTTP server
	s.http = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort),
		Handler: s.router,
	}

	// Start server
	s.logger.Printf("Starting server on port %s", cfg.ServerPort)
	return s.http.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.http != nil {
		return s.http.Shutdown(ctx)
	}
	return nil
}
