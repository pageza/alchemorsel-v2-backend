package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/mock"
)

// MockProfileService is a mock implementation of the ProfileService interface
type MockProfileService struct {
	mock.Mock
}

func (m *MockProfileService) GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockProfileService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *types.UpdateProfileRequest) (*models.UserProfile, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockProfileService) UpdateUserProfile(userID uuid.UUID, profile *models.UserProfile) error {
	args := m.Called(userID, profile)
	return args.Error(0)
}

func (m *MockProfileService) Logout(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockProfileService) GetProfileHistory(ctx context.Context, userID uuid.UUID) ([]*types.ProfileHistory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.ProfileHistory), args.Error(1)
}

func (m *MockProfileService) ValidateToken(token string) (*types.TokenClaims, error) {
	args := m.Called(token)
	return args.Get(0).(*types.TokenClaims), args.Error(1)
}

func (m *MockProfileService) GetUserRecipes(ctx context.Context, userID uuid.UUID) ([]*models.Recipe, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Recipe), args.Error(1)
}

func (m *MockProfileService) GetUserProfile(username string) (*models.UserProfile, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}
