package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/database"
	"github.com/pageza/alchemorsel-v2/backend/internal/server"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	migrationsDir := filepath.Join("migrations")
	if err := database.RunMigrations(db, migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create services
	authService := service.NewAuthService(db, cfg.JWTSecret)
	profileService := service.NewProfileService(db)

	// Create server
	srv := server.NewServer(db, authService, profileService)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(cfg.ServerPort); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Stop(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
