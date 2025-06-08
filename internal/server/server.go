package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: s.router,
	}

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.http != nil {
		return s.http.Shutdown(ctx)
	}
	return nil
}
