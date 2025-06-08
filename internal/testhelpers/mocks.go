package testhelpers

import (
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of the AuthService interface
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) ValidateToken(token string) (*types.TokenClaims, error) {
	args := m.Called(token)
	return args.Get(0).(*types.TokenClaims), args.Error(1)
}

// MockProfileService is a mock implementation of the ProfileService interface
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

func (m *MockProfileService) UpdateUserProfile(userID uuid.UUID, profile *models.UserProfile) error {
	args := m.Called(userID, profile)
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

func (m *MockProfileService) ValidateToken(token string) (*types.TokenClaims, error) {
	args := m.Called(token)
	return args.Get(0).(*types.TokenClaims), args.Error(1)
}

func (m *MockProfileService) GetUserRecipes(userID uuid.UUID) ([]models.Recipe, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Recipe), args.Error(1)
}

func (m *MockProfileService) GetUserProfile(userID uuid.UUID) (*models.UserProfile, error) {
	return &models.UserProfile{
		ID:       userID,
		UserID:   userID,
		Username: "testuser",
		Bio:      "Test bio",
	}, nil
}
