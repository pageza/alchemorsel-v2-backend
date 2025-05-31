package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBPassword: "postgres",
		DBName:     "alchemorsel_test",
		DBSSLMode:  "disable",
		JWTSecret:  "test-secret",
	}

	// Create server
	srv := New(cfg)
	assert.NotNil(t, srv)

	// Test health check endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/health", nil)
	srv.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
