package server

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/api"
	"github.com/pageza/alchemorsel-v2/backend/internal/database"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

// Server represents the HTTP server
type Server struct {
	config *config.Config
	http   *http.Server
	db     *database.DB
	router *gin.Engine
}

// New creates a new Server instance
func New(cfg *config.Config) *Server {
	router := gin.Default()
	srv := &Server{
		router: router,
		config: cfg,
	}

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		panic(err)
	}

	// Initialize services
	profileService := service.NewProfileService(db.GormDB, cfg.JWTSecret)

	// Register routes
	api.RegisterProfileRoutes(router, profileService)

	// Health check endpoint
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return srv
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting server on port %s", s.config.ServerPort)
	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.db != nil {
		s.db.Close()
	}
	return s.http.Shutdown(ctx)
}
