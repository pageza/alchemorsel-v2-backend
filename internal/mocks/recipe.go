package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/stretchr/testify/mock"
)

// MockRecipeService is a mock implementation of the recipe service
type MockRecipeService struct {
	mock.Mock
}

// CreateRecipe mocks the CreateRecipe method
func (m *MockRecipeService) CreateRecipe(ctx context.Context, recipe *models.Recipe) (*models.Recipe, error) {
	args := m.Called(ctx, recipe)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Recipe), args.Error(1)
}

// GetRecipe mocks the GetRecipe method
func (m *MockRecipeService) GetRecipe(ctx context.Context, id uuid.UUID) (*models.Recipe, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Recipe), args.Error(1)
}

// UpdateRecipe mocks the UpdateRecipe method
func (m *MockRecipeService) UpdateRecipe(ctx context.Context, id uuid.UUID, recipe *models.Recipe) (*models.Recipe, error) {
	args := m.Called(ctx, id, recipe)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Recipe), args.Error(1)
}

// DeleteRecipe mocks the DeleteRecipe method
func (m *MockRecipeService) DeleteRecipe(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ListRecipes mocks the ListRecipes method
func (m *MockRecipeService) ListRecipes(ctx context.Context, userID uuid.UUID) ([]*models.Recipe, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Recipe), args.Error(1)
}

// SearchRecipes mocks the SearchRecipes method
func (m *MockRecipeService) SearchRecipes(ctx context.Context, query string) ([]*models.Recipe, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Recipe), args.Error(1)
}
