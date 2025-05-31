package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pageza/alchemorsel-v2/backend/config"
)

func TestNew(t *testing.T) {
	// Use port 0 to let the OS assign an available port
	cfg := &config.Config{
		ServerPort: "0",
	}

	// Create a new server
	srv := New(cfg)

	// The actual port will be assigned by the OS, so we just check that the address is set
	if srv.http.Addr != ":0" {
		t.Errorf("Server address = %v; want %v", srv.http.Addr, ":0")
	}

	// Start the server in a goroutine
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed to start: %v", err)
		}
	}()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("Server failed to shutdown: %v", err)
	}
}
