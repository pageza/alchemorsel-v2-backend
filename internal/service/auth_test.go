package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Create PostgreSQL container
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "pgvector/pgvector:pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %v", err)
		}
	})

	// Get container port
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	// Connect to database
	dsn := fmt.Sprintf("host=localhost port=%s user=test password=test dbname=test sslmode=disable", port.Port())
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Install pgvector extension
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		t.Fatalf("failed to install pgvector extension: %v", err)
	}

	// Create dietary preference type enum
	if err := db.Exec(`
		DO $$ BEGIN
			CREATE TYPE dietary_preference_type AS ENUM (
				'vegetarian',
				'vegan',
				'pescatarian',
				'gluten-free',
				'dairy-free',
				'nut-free',
				'soy-free',
				'egg-free',
				'shellfish-free',
				'custom'
			);
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;
	`).Error; err != nil {
		t.Fatalf("failed to create dietary preference type: %v", err)
	}

	// Auto-migrate schema
	if err := db.AutoMigrate(&models.User{}, &models.UserProfile{}, &models.DietaryPreference{}, &models.Allergen{}); err != nil {
		t.Fatalf("failed to auto-migrate schema: %v", err)
	}

	return db
}

func setupAuthTest(t *testing.T) (*gin.Engine, *gorm.DB, *service.AuthService) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)

	authSvc := service.NewAuthService(db, "test-secret")
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	router.POST("/api/v1/auth/register", func(c *gin.Context) {
		var req struct {
			Email              string   `json:"email"`
			Password           string   `json:"password"`
			Username           string   `json:"username"`
			DietaryPreferences []string `json:"dietary_preferences"`
			Allergies          []string `json:"allergies"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, err := authSvc.Register(c.Request.Context(), req.Email, req.Password, &types.UserPreferences{
			DietaryPrefs: req.DietaryPreferences,
			Allergies:    req.Allergies,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Get user profile for username
		var profile models.UserProfile
		if err := db.Where("user_id = ?", user.ID).First(&profile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Generate token
		token, err := authSvc.GenerateToken(&types.TokenClaims{
			UserID:   user.ID,
			Username: profile.Username,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id": user.ID,
			"token":   token,
		})
	})

	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, profile, err := authSvc.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Generate token
		token, err := authSvc.GenerateToken(&types.TokenClaims{
			UserID:   user.ID,
			Username: profile.Username,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id": user.ID,
			"token":   token,
		})
	})

	return router, db, authSvc
}

func TestRegisterMissingPrefs(t *testing.T) {
	router, db, authSvc := setupAuthTest(t)
	defer db.Migrator().DropTable(&models.User{}, &models.UserProfile{})

	// Test registration without preferences
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{
		"email": "t@example.com",
		"password": "password123",
		"username": "tester",
		"dietary_preferences": [],
		"allergies": []
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 got %d", w.Code)
	}

	var response struct {
		UserID uuid.UUID `json:"user_id"`
		Token  string    `json:"token"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to parse resp: %v", err)
	}

	// Verify user was created
	var user models.User
	if err := db.Where("email = ?", "t@example.com").First(&user).Error; err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify token claims
	claims, err := authSvc.ValidateToken(response.Token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected user ID %v, got %v", user.ID, claims.UserID)
	}
	if claims.Username != "t@example.com" {
		t.Errorf("expected username 't@example.com', got %s", claims.Username)
	}
}

func TestRegisterWithPrefs(t *testing.T) {
	router, db, _ := setupAuthTest(t)
	defer db.Migrator().DropTable(&models.User{}, &models.UserProfile{})

	// Test registration with preferences
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{
		"email": "t2@example.com",
		"password": "password123",
		"username": "tester2",
		"dietary_preferences": ["vegetarian", "gluten-free"],
		"allergies": ["peanuts", "shellfish"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 got %d", w.Code)
	}

	var response struct {
		UserID uuid.UUID `json:"user_id"`
		Token  string    `json:"token"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to parse resp: %v", err)
	}

	// Verify user was created
	var user models.User
	if err := db.Where("email = ?", "t2@example.com").First(&user).Error; err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Verify token claims
	claims, err := service.NewAuthService(db, "test-secret").ValidateToken(response.Token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected user ID %v, got %v", user.ID, claims.UserID)
	}
	if claims.Username != "vegetarian" {
		t.Errorf("expected username 'vegetarian', got %s", claims.Username)
	}
}

func TestLogin(t *testing.T) {
	router, db, authSvc := setupAuthTest(t)
	defer db.Migrator().DropTable(&models.User{}, &models.UserProfile{})

	// Register a user first
	user, err := authSvc.Register(nil, "t3@example.com", "password123", &types.UserPreferences{
		DietaryPrefs: []string{"vegetarian"},
		Allergies:    []string{},
	})
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	// Test login
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
		"email": "t3@example.com",
		"password": "password123"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 got %d", w.Code)
	}

	var response struct {
		UserID uuid.UUID `json:"user_id"`
		Token  string    `json:"token"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to parse resp: %v", err)
	}

	// Verify token claims
	claims, err := authSvc.ValidateToken(response.Token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("expected user ID %v, got %v", user.ID, claims.UserID)
	}
	if claims.Username != "vegetarian" {
		t.Errorf("expected username 'vegetarian', got %s", claims.Username)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	router, db, authSvc := setupAuthTest(t)
	defer db.Migrator().DropTable(&models.User{}, &models.UserProfile{})

	// Register a user first
	_, err := authSvc.Register(nil, "t4@example.com", "password123", &types.UserPreferences{
		DietaryPrefs: []string{"vegetarian"},
		Allergies:    []string{},
	})
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	// Test login with wrong password
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{
		"email": "t4@example.com",
		"password": "wrongpassword"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 got %d", w.Code)
	}

	var response struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to parse resp: %v", err)
	}

	if response.Error != "invalid credentials" {
		t.Errorf("expected error 'invalid credentials', got %s", response.Error)
	}
}

func TestRegister(t *testing.T) {
	router, db, _ := setupAuthTest(t)
	defer db.Migrator().DropTable(&models.User{}, &models.UserProfile{})

	// Test registration
	req := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{
		"email": "test@example.com",
		"password": "password123",
		"username": "testuser",
		"dietary_preferences": ["vegetarian"],
		"allergies": []
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		UserID uuid.UUID `json:"user_id"`
		Token  string    `json:"token"`
	}
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.NotEmpty(t, response.UserID)
}
