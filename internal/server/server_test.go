package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestServer(t *testing.T) {
	// Set dummy environment variables for testing
	os.Setenv("DEEPSEEK_API_KEY", "test-deepseek-key")
	os.Setenv("OPENAI_API_KEY", "test-openai-key")
	defer func() {
		os.Unsetenv("DEEPSEEK_API_KEY")
		os.Unsetenv("OPENAI_API_KEY")
	}()

	// Use the exported SetupTestDatabase from testhelpers package
	db := testhelpers.SetupTestDatabase(t)
	defer db.DB().Migrator().DropTable(&models.Recipe{}, &models.User{}, &models.UserProfile{})

	// Create services
	authService := service.NewAuthService(db.DB(), "test-secret")
	profileService := service.NewProfileService(db.DB())

	// Create server
	cfg := &config.Config{
		JWTSecret: "test-secret",
	}
	server := NewServer(db.DB(), authService, profileService, cfg)

	// Test server initialization
	err := server.Start("8080")
	assert.NoError(t, err)

	// Test server shutdown
	ctx := context.Background()
	err = server.Stop(ctx)
	assert.NoError(t, err)
}

func TestNewServer(t *testing.T) {
	// Set dummy environment variables for testing
	os.Setenv("DEEPSEEK_API_KEY", "test-deepseek-key")
	os.Setenv("OPENAI_API_KEY", "test-openai-key")
	defer func() {
		os.Unsetenv("DEEPSEEK_API_KEY")
		os.Unsetenv("OPENAI_API_KEY")
	}()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NotNil(t, db)

	authService := service.NewAuthService(db, "test-secret")
	profileService := service.NewProfileService(db)

	cfg := &config.Config{
		JWTSecret: "test-secret",
	}
	server := NewServer(db, authService, profileService, cfg)
	assert.NotNil(t, server)

	// Test health check endpoint (already registered by NewServer)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify response body contains expected health check data
	expected := `{"message":"Alchemorsel API is running","status":"healthy","version":"v1.0.0"}`
	assert.JSONEq(t, expected, w.Body.String())
}
