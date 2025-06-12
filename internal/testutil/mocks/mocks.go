package mocks

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of the auth service
type MockAuthService struct {
	mock.Mock
}

// Register mocks the Register method
func (m *MockAuthService) Register(ctx context.Context, email, password, username string, preferences *types.UserPreferences) (*models.User, error) {
	args := m.Called(ctx, email, password, username, preferences)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// Login mocks the Login method
func (m *MockAuthService) Login(ctx context.Context, email, password string) (*models.User, *models.UserProfile, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*models.User), args.Get(1).(*models.UserProfile), args.Error(2)
}

// GenerateToken mocks the GenerateToken method
func (m *MockAuthService) GenerateToken(claims *types.TokenClaims) (string, error) {
	args := m.Called(claims)
	return args.String(0), args.Error(1)
}

// ValidateToken mocks the ValidateToken method
func (m *MockAuthService) ValidateToken(token string) (*types.TokenClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TokenClaims), args.Error(1)
}

// CreateTestUserAndToken mocks the CreateTestUserAndToken method
func (m *MockAuthService) CreateTestUserAndToken(t *testing.T) (string, string) {
	args := m.Called(t)
	return args.String(0), args.String(1)
}

// MockRecipeService is a mock implementation of the RecipeService interface
type MockRecipeService struct {
	mock.Mock
}

func (m *MockRecipeService) CreateRecipe(ctx context.Context, recipe *models.Recipe) (*models.Recipe, error) {
	args := m.Called(ctx, recipe)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Recipe), args.Error(1)
}

func (m *MockRecipeService) GetRecipe(ctx context.Context, id uuid.UUID) (*models.Recipe, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Recipe), args.Error(1)
}

func (m *MockRecipeService) UpdateRecipe(ctx context.Context, id uuid.UUID, recipe *models.Recipe) (*models.Recipe, error) {
	args := m.Called(ctx, id, recipe)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Recipe), args.Error(1)
}

func (m *MockRecipeService) DeleteRecipe(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRecipeService) GetUserRecipes(ctx context.Context, userID uuid.UUID) ([]*models.Recipe, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Recipe), args.Error(1)
}

// MockProfileService is a mock implementation of the ProfileService interface
type MockProfileService struct {
	mock.Mock
}

func (m *MockProfileService) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockProfileService) UpdateUserProfile(ctx context.Context, userID uuid.UUID, profile *models.UserProfile) (*models.UserProfile, error) {
	args := m.Called(ctx, userID, profile)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

func (m *MockProfileService) GetUserRecipes(ctx context.Context, userID uuid.UUID) ([]*models.Recipe, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Recipe), args.Error(1)
}
