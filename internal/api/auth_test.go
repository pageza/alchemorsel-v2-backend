package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/testhelpers"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	db := testhelpers.SetupTestDB(t) // Initialize test database
	_ = db                           // If not used directly, suppress unused warning
	router := SetupTestRouter(t)

	// Test registration
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(`{
		"email": "test@example.com",
		"password": "testpassword123",
		"username": "testuser"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["user_id"])
}

func TestLogin(t *testing.T) {
	db := testhelpers.SetupTestDB(t)
	authService := service.NewAuthService(db.DB(), "test-secret")
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user, profile, err := authService.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		token, err := authService.GenerateToken(&types.TokenClaims{
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

	// Test login
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(`{
		"email": "test@example.com",
		"password": "testpassword123"
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
