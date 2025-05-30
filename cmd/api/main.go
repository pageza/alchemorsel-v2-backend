package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/server"
)

func main() {
	// Initialize configuration
	cfg := config.New()

	// Create and start server
	srv := server.New(cfg)

	// Channel to listen for errors coming from the server
	errChan := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Println("Starting server...")
		errChan <- srv.Start()
	}()

	// Channel to listen for an interrupt or terminate signal from the OS
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal or error
	select {
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case sig := <-quit:
		log.Printf("Received signal: %v", sig)
	}

	// Gracefully shutdown the server
	log.Println("Shutting down server...")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
