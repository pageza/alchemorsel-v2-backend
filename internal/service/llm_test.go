package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLLMService(t *testing.T) {
	// Set up test environment
	originalKey := os.Getenv("DEEPSEEK_API_KEY")
	originalRedisHost := os.Getenv("REDIS_HOST")
	
	defer func() {
		os.Setenv("DEEPSEEK_API_KEY", originalKey)
		os.Setenv("REDIS_HOST", originalRedisHost)
	}()

	t.Run("should create service with API key", func(t *testing.T) {
		os.Setenv("DEEPSEEK_API_KEY", "test-api-key")
		os.Setenv("REDIS_HOST", "localhost")
		
		service, err := NewLLMService()
		
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.NotNil(t, service.client)
		assert.NotNil(t, service.redis)
		assert.NotNil(t, service.jsonExtractor)
	})

	t.Run("should fail without API key", func(t *testing.T) {
		os.Unsetenv("DEEPSEEK_API_KEY")
		os.Unsetenv("DEEPSEEK_API_KEY_FILE")
		
		service, err := NewLLMService()
		
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "DEEPSEEK_API_KEY or DEEPSEEK_API_KEY_FILE must be set")
	})
}

func TestLLMService_SaveDraft(t *testing.T) {
	// Skip this test if no Redis is available
	if os.Getenv("REDIS_HOST") == "" {
		t.Skip("Skipping Redis-dependent test - REDIS_HOST not set")
	}

	os.Setenv("DEEPSEEK_API_KEY", "test-api-key")
	service, err := NewLLMService()
	require.NoError(t, err)

	draft := &RecipeDraft{
		Name:         "Test Recipe",
		Description:  "A test recipe",
		Category:     "Main Course",
		Ingredients:  []string{"ingredient1", "ingredient2"},
		Instructions: []string{"step1", "step2"},
		PrepTime:     "15 minutes",
		CookTime:     "30 minutes",
		Servings:     ServingsType{Value: "4"},
		Difficulty:   "Easy",
		Calories:     300,
		Protein:      20,
		Carbs:        30,
		Fat:          10,
		UserID:       "test-user",
	}

	t.Run("should save and retrieve draft", func(t *testing.T) {
		ctx := testContext()
		
		err := service.SaveDraft(ctx, draft)
		require.NoError(t, err)
		assert.NotEmpty(t, draft.ID)
		assert.False(t, draft.CreatedAt.IsZero())
		assert.False(t, draft.UpdatedAt.IsZero())

		retrieved, err := service.GetDraft(ctx, draft.ID)
		require.NoError(t, err)
		assert.Equal(t, draft.Name, retrieved.Name)
		assert.Equal(t, draft.Description, retrieved.Description)
		assert.Equal(t, draft.Category, retrieved.Category)
		assert.Equal(t, draft.Ingredients, retrieved.Ingredients)
		assert.Equal(t, draft.Instructions, retrieved.Instructions)

		// Clean up
		err = service.DeleteDraft(ctx, draft.ID)
		assert.NoError(t, err)
	})
}

func TestLLMService_UpdateDraft(t *testing.T) {
	// Skip this test if no Redis is available
	if os.Getenv("REDIS_HOST") == "" {
		t.Skip("Skipping Redis-dependent test - REDIS_HOST not set")
	}

	os.Setenv("DEEPSEEK_API_KEY", "test-api-key")
	service, err := NewLLMService()
	require.NoError(t, err)

	draft := &RecipeDraft{
		Name:         "Original Recipe",
		Description:  "Original description",
		Category:     "Main Course",
		Ingredients:  []string{"ingredient1"},
		Instructions: []string{"step1"},
		PrepTime:     "15 minutes",
		CookTime:     "30 minutes",
		Servings:     ServingsType{Value: "4"},
		Difficulty:   "Easy",
		Calories:     300,
		Protein:      20,
		Carbs:        30,
		Fat:          10,
		UserID:       "test-user",
	}

	t.Run("should update existing draft", func(t *testing.T) {
		ctx := testContext()
		
		// Save original
		err := service.SaveDraft(ctx, draft)
		require.NoError(t, err)
		
		originalUpdatedAt := draft.UpdatedAt
		
		// Wait a moment to ensure timestamp difference
		time.Sleep(10 * time.Millisecond)
		
		// Update
		draft.Name = "Updated Recipe"
		draft.Description = "Updated description"
		err = service.UpdateDraft(ctx, draft)
		require.NoError(t, err)
		
		// Verify update
		assert.True(t, draft.UpdatedAt.After(originalUpdatedAt))
		
		retrieved, err := service.GetDraft(ctx, draft.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Recipe", retrieved.Name)
		assert.Equal(t, "Updated description", retrieved.Description)

		// Clean up
		err = service.DeleteDraft(ctx, draft.ID)
		assert.NoError(t, err)
	})
}

func TestRecipeData_Validation(t *testing.T) {
	t.Run("should have required fields", func(t *testing.T) {
		recipe := RecipeData{
			Name:         "Test Recipe",
			Description:  "A test recipe",
			Category:     "Main Course",
			Cuisine:      "Italian",
			Ingredients:  []string{"ingredient1", "ingredient2"},
			Instructions: []string{"step1", "step2"},
			PrepTime:     "15 minutes",
			CookTime:     "30 minutes",
			Servings:     "4",
			Difficulty:   "Easy",
		}

		assert.NotEmpty(t, recipe.Name)
		assert.NotEmpty(t, recipe.Description)
		assert.NotEmpty(t, recipe.Category)
		assert.NotEmpty(t, recipe.Cuisine)
		assert.NotEmpty(t, recipe.Ingredients)
		assert.NotEmpty(t, recipe.Instructions)
		assert.NotEmpty(t, recipe.PrepTime)
		assert.NotEmpty(t, recipe.CookTime)
		assert.NotEmpty(t, recipe.Servings)
		assert.NotEmpty(t, recipe.Difficulty)
	})
}

func TestServingsType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "number input",
			input:    `4`,
			expected: "4",
		},
		{
			name:     "string input",
			input:    `"6 servings"`,
			expected: "6 servings",
		},
		{
			name:     "object input",
			input:    `{"Value": "8"}`,
			expected: "8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var servings ServingsType
			err := servings.UnmarshalJSON([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, servings.Value)
		})
	}

	t.Run("should handle invalid input", func(t *testing.T) {
		var servings ServingsType
		err := servings.UnmarshalJSON([]byte(`invalid`))
		assert.Error(t, err)
	})
}

// testContext returns a test context with timeout
func testContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return ctx
}