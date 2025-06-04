package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/middleware"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockProfileService struct {
	mock.Mock
}

func (m *MockProfileService) GetProfile(userID uuid.UUID) (*models.UserProfile, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockProfileService) UpdateProfile(userID uuid.UUID, updates map[string]interface{}) error {
	args := m.Called(userID, updates)
	return args.Error(0)
}

func (m *MockProfileService) Logout(userID uuid.UUID) error {
	args := m.Called(userID)
	return args.Error(0)
}

func (m *MockProfileService) GetProfileHistory(userID uuid.UUID) ([]map[string]interface{}, error) {
	args := m.Called(userID)
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockProfileService) ValidateToken(token string) (*middleware.TokenClaims, error) {
	args := m.Called(token)
	return args.Get(0).(*middleware.TokenClaims), args.Error(1)
}

func TestGetProfile(t *testing.T) {
	mockService := new(MockProfileService)
	handler := NewProfileHandler(mockService)

	// Create a test UUID
	testUUID := uuid.New()

	// Mock data
	expectedProfile := &models.UserProfile{
		ID:       testUUID,
		UserID:   testUUID,
		Username: "testuser",
		Bio:      "Test bio",
	}

	// Setup mock expectations
	mockService.On("GetProfile", testUUID).Return(expectedProfile, nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", testUUID)

	// Call handler
	handler.GetProfile(c)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestUpdateProfile(t *testing.T) {
	mockService := new(MockProfileService)
	handler := NewProfileHandler(mockService)

	// Create a test UUID
	testUUID := uuid.New()

	// Test request body
	requestBody := []byte(`{"username": "newusername", "bio": "New bio"}`)

	// Setup mock expectations
	mockService.On("UpdateProfile", testUUID, mock.Anything).Return(nil)

	// Create test request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", testUUID)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewBuffer(requestBody))

	// Call handler
	handler.UpdateProfile(c)

	// Assert response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}
