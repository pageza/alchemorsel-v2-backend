package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/testhelpers/mocks"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/mock"
)

func TestGetProfile(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockAuthService := new(mocks.MockAuthService)
	mockProfileService := new(mocks.MockProfileService)
	profileHandler := NewProfileHandler(mockProfileService, mockAuthService)

	// Create a test UUID
	testUUID := uuid.New()

	// Mock token validation
	claims := &types.TokenClaims{
		UserID:   testUUID,
		Username: "testuser",
	}
	mockAuthService.On("ValidateToken", mock.Anything).Return(claims, nil)

	// Mock data
	expectedProfile := &models.UserProfile{
		ID:       uuid.New(),
		UserID:   testUUID,
		Username: "testuser",
		Bio:      "Test bio",
	}

	// Mock recipes
	expectedRecipes := []*models.Recipe{
		{ID: uuid.New(), Name: "Test", Ingredients: []string{}, Instructions: []string{}, UserID: testUUID},
	}

	// Setup mock expectations
	mockProfileService.On("GetProfile", context.Background(), testUUID).Return(expectedProfile, nil)
	mockProfileService.On("GetUserRecipes", context.Background(), testUUID).Return(expectedRecipes, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/profile", nil)
	c.Set("user_id", testUUID.String())
	c.Set("username", "testuser")

	// Call handler
	profileHandler.GetProfile(c)

	// Assert response
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var resp struct {
		Profile models.UserProfile `json:"profile"`
		Recipes []models.Recipe    `json:"recipes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Recipes) != len(expectedRecipes) {
		t.Fatalf("expected %d recipes got %d", len(expectedRecipes), len(resp.Recipes))
	}
}

func TestUpdateProfile(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockAuthService := new(mocks.MockAuthService)
	mockProfileService := new(mocks.MockProfileService)
	profileHandler := NewProfileHandler(mockProfileService, mockAuthService)

	// Create a test UUID
	testUUID := uuid.New()

	// Mock token validation
	claims := &types.TokenClaims{
		UserID:   testUUID,
		Username: "testuser",
	}
	mockAuthService.On("ValidateToken", mock.Anything).Return(claims, nil)

	// Set up the mock expectation
	bio := "New bio"
	expectedProfile := &models.UserProfile{
		ID:       uuid.New(),
		UserID:   testUUID,
		Username: "newusername",
		Bio:      bio,
	}
	mockProfileService.On("UpdateProfile", context.Background(), testUUID, mock.Anything).Return(expectedProfile, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create test request
	req := types.UpdateProfileRequest{
		Username: "newusername",
		Bio:      &bio,
	}
	jsonData, _ := json.Marshal(req)
	c.Request = httptest.NewRequest(http.MethodPut, "/profile", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", testUUID.String())
	c.Set("username", "testuser")

	profileHandler.UpdateProfile(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, w.Code)
	}

	mockProfileService.AssertExpectations(t)
}
