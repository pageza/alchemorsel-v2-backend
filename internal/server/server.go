package server

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/api"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

// Server represents the HTTP server
type Server struct {
	router  *gin.Engine
	http    *http.Server
	db      *gorm.DB
	logger  *log.Logger
	auth    *service.AuthService
	profile *service.ProfileService
}

// NewServer creates a new server instance
func NewServer(db *gorm.DB, auth *service.AuthService, profile *service.ProfileService) *Server {
	router := gin.Default()

	// Add CORS middleware
	router.Use(middleware.CORS())

	// Create API handlers
	authHandler := api.NewAuthHandler(auth, db)

	// Register routes
	api.RegisterProfileRoutes(router, profile, auth)
	authHandler.RegisterRoutes(router.Group("/api/v1"))

	return &Server{
		router:  router,
		db:      db,
		auth:    auth,
		profile: profile,
	}
}

// Start starts the server
func (s *Server) Start(port string) error {
	s.http = &http.Server{
		Addr:    ":" + port,
		Handler: s.router,
	}

	// Start server in a goroutine
	go func() {
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.http != nil {
		return s.http.Shutdown(ctx)
	}
	return nil
}
