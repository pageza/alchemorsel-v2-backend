package main

import (
	"log"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/database"
	"github.com/pageza/alchemorsel-v2/backend/internal/server"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize services
	authService := service.NewAuthService(db, cfg.JWTSecret)
	profileService := service.NewProfileService(db)

	// Create and start server
	srv := server.NewServer(db, authService, profileService, cfg)
	if err := srv.Start(cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
