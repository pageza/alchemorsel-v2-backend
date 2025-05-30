package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/database"
)

// Server represents the HTTP server
type Server struct {
	config *config.Config
	http   *http.Server
	db     *database.DB
}

// New creates a new Server instance
func New(cfg *config.Config) *Server {
	// Initialize database connection
	db, err := database.New(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
	}

	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		health := map[string]string{
			"status": "ok",
		}

		// Check database health if connection exists
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			if err := db.HealthCheck(ctx); err != nil {
				health["db"] = "unreachable"
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				health["db"] = "ok"
			}
		} else {
			health["db"] = "not_initialized"
		}

		json.NewEncoder(w).Encode(health)
	})

	return &Server{
		config: cfg,
		db:     db,
		http: &http.Server{
			Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}
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
