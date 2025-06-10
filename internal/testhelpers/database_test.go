package testhelpers

import (
	"testing"

	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	pgvector "github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
)

func TestDatabaseSetup(t *testing.T) {
	// Use the exported SetupTestDB from testhelpers package
	db := SetupTestDB(t)
	assert.NotNil(t, db)

	// Test creating a user with profile
	user := &models.User{
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		Profile: models.UserProfile{
			Username: "testuser",
			Bio:      "Test bio",
		},
	}
	err := db.DB().Create(user).Error
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)
	assert.NotZero(t, user.Profile.ID)

	// Test creating dietary preferences
	dietaryPref := &models.DietaryPreference{
		UserID:         user.ID,
		PreferenceType: "vegetarian",
		CustomName:     "Vegetarian",
	}
	err = db.DB().Create(dietaryPref).Error
	assert.NoError(t, err)
	assert.NotZero(t, dietaryPref.ID)

	// Test creating allergens
	allergen := &models.Allergen{
		UserID:        user.ID,
		AllergenName:  "peanuts",
		SeverityLevel: 1,
	}
	err = db.DB().Create(allergen).Error
	assert.NoError(t, err)
	assert.NotZero(t, allergen.ID)

	// Test creating a recipe
	recipe := &models.Recipe{
		Name:               "Test Recipe",
		Description:        "Test Description",
		UserID:             user.ID,
		Category:           "Main Course",
		Cuisine:            "Italian",
		Ingredients:        []string{"ingredient1", "ingredient2"},
		Instructions:       []string{"step1", "step2"},
		Calories:           100,
		Protein:            10,
		Carbs:              20,
		Fat:                5,
		Embedding:          pgvector.NewVector([]float32{1.0, 2.0, 3.0}), // Example embedding
		DietaryPreferences: []string{"vegetarian"},
		Tags:               []string{"healthy", "quick"},
	}
	err = db.DB().Create(recipe).Error
	assert.NoError(t, err)
	assert.NotZero(t, recipe.ID)

	// Test creating a recipe favorite
	favorite := &models.RecipeFavorite{
		RecipeID: recipe.ID,
		UserID:   user.ID,
	}
	err = db.DB().Create(favorite).Error
	assert.NoError(t, err)
	assert.NotZero(t, favorite.ID)

	// Test loading user with relationships
	var loadedUser models.User
	err = db.DB().Preload("Profile").
		Preload("DietaryPrefs").
		Preload("Allergens").
		First(&loadedUser, user.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, user.ID, loadedUser.ID)
	assert.Equal(t, user.Profile.Username, loadedUser.Profile.Username)
	assert.Len(t, loadedUser.DietaryPrefs, 1)
	assert.Len(t, loadedUser.Allergens, 1)

	// Test loading recipe with relationships
	var loadedRecipe models.Recipe
	err = db.DB().First(&loadedRecipe, recipe.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, recipe.ID, loadedRecipe.ID)
	assert.Equal(t, recipe.Name, loadedRecipe.Name)
	assert.Len(t, loadedRecipe.Ingredients, 2)
	assert.Len(t, loadedRecipe.Instructions, 2)
	assert.Len(t, loadedRecipe.DietaryPreferences, 1)
	assert.Len(t, loadedRecipe.Tags, 2)

	// Test loading recipe favorite
	var loadedFavorite models.RecipeFavorite
	err = db.DB().First(&loadedFavorite, favorite.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, favorite.ID, loadedFavorite.ID)
	assert.Equal(t, recipe.ID, loadedFavorite.RecipeID)
	assert.Equal(t, user.ID, loadedFavorite.UserID)
}
