package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/stretchr/testify/mock"
)

// MockEmbeddingService is a mock implementation of the embedding service
type MockEmbeddingService struct{}

func (m *MockEmbeddingService) GenerateEmbedding(text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *MockEmbeddingService) GenerateEmbeddingFromRecipe(recipe *types.Recipe) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

// MockLLMService is a mock implementation of the LLM service
type MockLLMService struct {
	drafts map[string]*service.RecipeDraft
}

func NewMockLLMService() *MockLLMService {
	return &MockLLMService{
		drafts: make(map[string]*service.RecipeDraft),
	}
}

func (m *MockLLMService) GenerateRecipe(query string, dietaryPrefs, allergens []string, originalRecipe *service.RecipeDraft) (string, error) {
	return `{"name":"Test Recipe","description":"Desc","category":"Cat","ingredients":["i1"],"instructions":["s1"],"calories":100,"protein":10,"carbs":20,"fat":5}`, nil
}

func (m *MockLLMService) SaveDraft(ctx context.Context, draft *service.RecipeDraft) error {
	draft.ID = "test-draft-id"
	m.drafts[draft.ID] = draft
	return nil
}

func (m *MockLLMService) GetDraft(ctx context.Context, id string) (*service.RecipeDraft, error) {
	if draft, exists := m.drafts[id]; exists {
		return draft, nil
	}
	return &service.RecipeDraft{
		ID:           id,
		Name:         "Test Recipe",
		Description:  "Desc",
		Category:     "Cat",
		Ingredients:  []string{"i1"},
		Instructions: []string{"s1"},
		Calories:     100,
		Protein:      10,
		Carbs:        20,
		Fat:          5,
	}, nil
}

func (m *MockLLMService) UpdateDraft(ctx context.Context, draft *service.RecipeDraft) error {
	m.drafts[draft.ID] = draft
	return nil
}

func (m *MockLLMService) DeleteDraft(ctx context.Context, id string) error {
	delete(m.drafts, id)
	return nil
}

func (m *MockLLMService) CalculateMacros(ingredients []string) (*service.Macros, error) {
	return &service.Macros{
		Calories: 100,
		Protein:  10,
		Carbs:    20,
		Fat:      5,
	}, nil
}

func (m *MockLLMService) GenerateRecipesBatch(prompts []string) ([]string, error) {
	return []string{`{"name":"Test Recipe","description":"Desc","category":"Cat","ingredients":["i1"],"instructions":["s1"],"calories":100,"protein":10,"carbs":20,"fat":5}`}, nil
}

// MockTokenValidator is a mock token validator for testing
type MockTokenValidator struct {
	mock.Mock
}

// ValidateToken validates a token and returns claims
func (v *MockTokenValidator) ValidateToken(token string) (*types.TokenClaims, error) {
	args := v.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TokenClaims), args.Error(1)
}

// MockAuthService is a mock implementation of the auth service
type MockAuthService struct {
	mock.Mock
}

// Register mocks the Register method
func (m *MockAuthService) Register(ctx context.Context, email, password string, preferences *types.UserPreferences) (*models.User, error) {
	args := m.Called(ctx, email, password, preferences)
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

// ValidateToken mocks the ValidateToken method
func (m *MockAuthService) ValidateToken(token string) (*types.TokenClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.TokenClaims), args.Error(1)
}

// GenerateToken mocks the GenerateToken method
func (m *MockAuthService) GenerateToken(claims *types.TokenClaims) (string, error) {
	args := m.Called(claims)
	return args.String(0), args.Error(1)
}

// MockProfileService is a mock implementation of the profile service
type MockProfileService struct {
	mock.Mock
}

// GetProfile mocks the GetProfile method
func (m *MockProfileService) GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

// UpdateProfile mocks the UpdateProfile method
func (m *MockProfileService) UpdateProfile(ctx context.Context, userID uuid.UUID, profile *types.UpdateProfileRequest) (*models.UserProfile, error) {
	args := m.Called(ctx, userID, profile)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

// Logout mocks the Logout method
func (m *MockProfileService) Logout(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// GetUserRecipes mocks the GetUserRecipes method
func (m *MockProfileService) GetUserRecipes(ctx context.Context, userID uuid.UUID) ([]*models.Recipe, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Recipe), args.Error(1)
}

// GetProfileHistory mocks the GetProfileHistory method
func (m *MockProfileService) GetProfileHistory(ctx context.Context, userID uuid.UUID) ([]*types.ProfileHistory, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*types.ProfileHistory), args.Error(1)
}

// GetUserProfile mocks the GetUserProfile method
func (m *MockProfileService) GetUserProfile(userID string) (*models.UserProfile, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserProfile), args.Error(1)
}

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
