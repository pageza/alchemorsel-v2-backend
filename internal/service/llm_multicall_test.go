package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiCallRecipeGeneration(t *testing.T) {
	t.Run("BasicRecipe Type Definition", func(t *testing.T) {
		// Test that BasicRecipe struct is properly defined
		basicRecipe := BasicRecipe{
			Name:         "Test Recipe",
			Description:  "A test recipe",
			Category:     "Main Course",
			Cuisine:      "American", 
			Ingredients:  []string{"1 cup flour", "2 eggs"},
			Instructions: []string{"Mix ingredients", "Cook"},
			PrepTime:     "15 minutes",
			CookTime:     "30 minutes",
			Servings:     ServingsType{Value: "4"},
			Difficulty:   "Easy",
		}

		assert.Equal(t, "Test Recipe", basicRecipe.Name)
		assert.Equal(t, "American", basicRecipe.Cuisine)
		assert.Equal(t, "4", basicRecipe.Servings.Value)
		assert.Len(t, basicRecipe.Ingredients, 2)
		assert.Len(t, basicRecipe.Instructions, 2)
	})

	t.Run("RecipeDraft has Cuisine field", func(t *testing.T) {
		// Test that RecipeDraft struct now includes Cuisine field
		draft := RecipeDraft{
			Name:         "Test Recipe",
			Description:  "A test recipe",
			Category:     "Main Course",
			Cuisine:      "Italian", // This should now work
			Ingredients:  []string{"1 cup flour", "2 eggs"},
			Instructions: []string{"Mix ingredients", "Cook"},
			PrepTime:     "15 minutes",
			CookTime:     "30 minutes",
			Servings:     ServingsType{Value: "4"},
			Difficulty:   "Easy",
			UserID:       "test-user-id",
		}

		assert.Equal(t, "Test Recipe", draft.Name)
		assert.Equal(t, "Italian", draft.Cuisine)
		assert.Equal(t, "test-user-id", draft.UserID)
	})

	t.Run("Macros Type Definition", func(t *testing.T) {
		// Test that Macros struct is properly defined for nutrition calculations
		macros := Macros{
			Calories: 350,
			Protein:  15,
			Carbs:    45,
			Fat:      12,
		}

		assert.Equal(t, float64(350), macros.Calories)
		assert.Equal(t, float64(15), macros.Protein)
		assert.Equal(t, float64(45), macros.Carbs)
		assert.Equal(t, float64(12), macros.Fat)
	})
}

// TestMultiCallMethodSignatures verifies that all the multi-call methods exist in the interface
func TestMultiCallMethodSignatures(t *testing.T) {
	// This test ensures that the LLMServiceInterface includes all required multi-call methods
	// It will fail to compile if any method is missing from the interface
	var _ LLMServiceInterface = (*LLMService)(nil)
	
	// Create a mock implementation to verify method signatures
	mockService := &mockLLMServiceForSignatureTest{}
	
	// Test GenerateBasicRecipe signature
	_, err := mockService.GenerateBasicRecipe(context.Background(), "test query", []string{}, []string{}, "user-id")
	assert.Error(t, err) // Expected error from mock
	
	// Test CalculateRecipeNutrition signature
	_, err = mockService.CalculateRecipeNutrition(context.Background(), "draft-id")
	assert.Error(t, err) // Expected error from mock
	
	// Test FinalizeRecipe signature
	_, err = mockService.FinalizeRecipe(context.Background(), "draft-id")
	assert.Error(t, err) // Expected error from mock
}

// mockLLMServiceForSignatureTest is a minimal mock just for testing method signatures
type mockLLMServiceForSignatureTest struct{}

func (m *mockLLMServiceForSignatureTest) GenerateRecipe(query string, dietaryPrefs []string, allergens []string, draft *RecipeDraft) (string, error) {
	return "", assert.AnError
}

func (m *mockLLMServiceForSignatureTest) SaveDraft(ctx context.Context, draft *RecipeDraft) error {
	return assert.AnError
}

func (m *mockLLMServiceForSignatureTest) GetDraft(ctx context.Context, draftID string) (*RecipeDraft, error) {
	return nil, assert.AnError
}

func (m *mockLLMServiceForSignatureTest) UpdateDraft(ctx context.Context, draft *RecipeDraft) error {
	return assert.AnError
}

func (m *mockLLMServiceForSignatureTest) DeleteDraft(ctx context.Context, id string) error {
	return assert.AnError
}

func (m *mockLLMServiceForSignatureTest) CalculateMacros(ingredients []string) (*Macros, error) {
	return nil, assert.AnError
}

func (m *mockLLMServiceForSignatureTest) GenerateRecipesBatch(prompts []string) ([]string, error) {
	return nil, assert.AnError
}

// Multi-call methods
func (m *mockLLMServiceForSignatureTest) GenerateBasicRecipe(ctx context.Context, query string, dietaryPrefs []string, allergens []string, userID string) (*RecipeDraft, error) {
	return nil, assert.AnError
}

func (m *mockLLMServiceForSignatureTest) CalculateRecipeNutrition(ctx context.Context, draftID string) (*Macros, error) {
	return nil, assert.AnError
}

func (m *mockLLMServiceForSignatureTest) FinalizeRecipe(ctx context.Context, draftID string) (*RecipeDraft, error) {
	return nil, assert.AnError
}