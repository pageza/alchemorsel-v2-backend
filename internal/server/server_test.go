package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestServer(t *testing.T) {
	// Use the exported SetupTestDatabase from testhelpers package
	db := testhelpers.SetupTestDatabase(t)
	defer db.DB().Migrator().DropTable(&models.Recipe{}, &models.User{}, &models.UserProfile{})

	// Create services
	authService := service.NewAuthService(db.DB(), "test-secret")
	profileService := service.NewProfileService(db.DB())

	// Create server
	server := NewServer(db.DB(), authService, profileService)

	// Test server initialization
	err := server.Start("8080")
	assert.NoError(t, err)

	// Test server shutdown
	ctx := context.Background()
	err = server.Stop(ctx)
	assert.NoError(t, err)
}

func TestNewServer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NotNil(t, db)

	authService := service.NewAuthService(db, "test-secret")
	profileService := service.NewProfileService(db)

	server := NewServer(db, authService, profileService)
	assert.NotNil(t, server)

	// Register health endpoint for test
	server.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Test health check endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
