package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockProfileService struct {
	mock.Mock
}

func (m *MockProfileService) GetProfile(userID uint) (*models.Profile, error) {
	args := m.Called(userID)
	return args.Get(0).(*models.Profile), args.Error(1)
}

func (m *MockProfileService) UpdateProfile(userID uint, profile *models.Profile) error {
	args := m.Called(userID, profile)
	return args.Error(0)
}

func (m *MockProfileService) GetProfileHistory(userID uint) ([]map[string]interface{}, error) {
	args := m.Called(userID)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockProfileService) ValidateToken(token string) (*middleware.TokenClaims, error) {
	args := m.Called(token)
	return args.Get(0).(*middleware.TokenClaims), args.Error(1)
}

func TestGetProfile(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockService := new(MockProfileService)
	handler := NewProfileHandler(mockService)

	// Mock data
	expectedProfile := &models.Profile{
		UserID:   1,
		Username: "testuser",
		Email:    "test@example.com",
	}

	// Setup mock expectations
	mockService.On("GetProfile", uint(1)).Return(expectedProfile, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(1))

	// Call handler
	handler.GetProfile(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "testuser", response["username"])
	assert.Equal(t, "test@example.com", response["email"])

	mockService.AssertExpectations(t)
}

func TestUpdateProfile(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockService := new(MockProfileService)
	handler := NewProfileHandler(mockService)

	// Mock data
	updateData := map[string]interface{}{
		"username": "updateduser",
		"email":    "updated@example.com",
	}
	requestBody, _ := json.Marshal(updateData)

	// Setup mock expectations
	mockService.On("UpdateProfile", uint(1), mock.AnythingOfType("*models.Profile")).Return(nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", uint(1))
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewBuffer(requestBody))

	// Call handler
	handler.UpdateProfile(c)

	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Profile updated successfully", response["message"])

	mockService.AssertExpectations(t)
}
