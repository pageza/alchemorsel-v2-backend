package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TestJSONStringArray is a custom type for handling string arrays in SQLite
type TestJSONStringArray []string

// Value implements the driver.Valuer interface
func (a TestJSONStringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface
func (a *TestJSONStringArray) Scan(value interface{}) error {
	if value == nil {
		*a = TestJSONStringArray{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value: %v", value)
	}
	return json.Unmarshal(bytes, a)
}

// TestRecipe represents a recipe in the test database
type TestRecipe struct {
	ID                 string              `gorm:"primarykey" json:"id"`
	UserID             string              `gorm:"not null" json:"user_id"`
	Name               string              `gorm:"not null" json:"name"`
	Description        string              `json:"description"`
	Category           string              `json:"category"`
	Cuisine            string              `json:"cuisine"`
	ImageURL           string              `json:"image_url"`
	Ingredients        TestJSONStringArray `json:"ingredients"`
	Instructions       TestJSONStringArray `json:"instructions"`
	Calories           float64             `json:"calories"`
	Protein            float64             `json:"protein"`
	Carbs              float64             `json:"carbs"`
	Fat                float64             `json:"fat"`
	DietaryPreferences TestJSONStringArray `json:"dietary_preferences"`
	Tags               TestJSONStringArray `json:"tags"`
	Embedding          []float32           `gorm:"type:text" json:"embedding"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	DeletedAt          gorm.DeletedAt      `gorm:"index" json:"-"`
}

// Value implements the driver.Valuer interface for TestRecipe.Embedding
func (r TestRecipe) Value() (driver.Value, error) {
	if r.Embedding == nil {
		return nil, nil
	}
	return json.Marshal(r.Embedding)
}

// Scan implements the sql.Scanner interface for TestRecipe.Embedding
func (r *TestRecipe) Scan(value interface{}) error {
	if value == nil {
		r.Embedding = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal embedding value: %v", value)
	}
	return json.Unmarshal(bytes, &r.Embedding)
}

// TestRecipeFavorite represents a recipe favorite in the test database
type TestRecipeFavorite struct {
	ID        string         `gorm:"primarykey" json:"id"`
	UserID    string         `gorm:"not null" json:"user_id"`
	RecipeID  string         `gorm:"not null" json:"recipe_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TestUser represents a user in the test database
type TestUser struct {
	ID           string         `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"not null" json:"name"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
}

// TestUserPreference represents a user preference in the test database
type TestUserPreference struct {
	ID        string              `gorm:"primarykey" json:"id"`
	UserID    string              `gorm:"uniqueIndex;not null" json:"user_id"`
	Dietary   TestJSONStringArray `json:"dietary"`
	Allergies TestJSONStringArray `json:"allergies"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	DeletedAt gorm.DeletedAt      `gorm:"index" json:"-"`
}

// API Request/Response Types

// CreateRecipeRequest is used for creating a recipe in tests
type CreateRecipeRequest struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Category           string   `json:"category"`
	Cuisine            string   `json:"cuisine"`
	ImageURL           string   `json:"image_url"`
	Ingredients        []string `json:"ingredients"`
	Instructions       []string `json:"instructions"`
	Calories           float64  `json:"calories"`
	Protein            float64  `json:"protein"`
	Carbs              float64  `json:"carbs"`
	Fat                float64  `json:"fat"`
	DietaryPreferences []string `json:"dietary_preferences"`
	Tags               []string `json:"tags"`
}

// UpdateRecipeRequest is used for updating a recipe in tests
type UpdateRecipeRequest struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Category           string   `json:"category"`
	Cuisine            string   `json:"cuisine"`
	ImageURL           string   `json:"image_url"`
	Ingredients        []string `json:"ingredients"`
	Instructions       []string `json:"instructions"`
	Calories           float64  `json:"calories"`
	Protein            float64  `json:"protein"`
	Carbs              float64  `json:"carbs"`
	Fat                float64  `json:"fat"`
	DietaryPreferences []string `json:"dietary_preferences"`
	Tags               []string `json:"tags"`
}

// RecipeResponse is used for recipe API responses in tests
type RecipeResponse struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Category           string   `json:"category"`
	Cuisine            string   `json:"cuisine"`
	ImageURL           string   `json:"image_url"`
	Ingredients        []string `json:"ingredients"`
	Instructions       []string `json:"instructions"`
	Calories           float64  `json:"calories"`
	Protein            float64  `json:"protein"`
	Carbs              float64  `json:"carbs"`
	Fat                float64  `json:"fat"`
	DietaryPreferences []string `json:"dietary_preferences"`
	Tags               []string `json:"tags"`
	UserID             string   `json:"user_id"`
}

// ListRecipesResponse is used for listing recipes in tests
type ListRecipesResponse struct {
	Recipes []RecipeResponse `json:"recipes"`
}

// RegisterRequest is used for user registration in tests
type RegisterRequest struct {
	Name               string   `json:"name"`
	Email              string   `json:"email"`
	Password           string   `json:"password"`
	Username           string   `json:"username"`
	DietaryPreferences []string `json:"dietary_preferences"`
	Allergies          []string `json:"allergies"`
}

// LoginRequest is used for user login in tests
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse is used for login API responses in tests
type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
}

// ProfileResponse is used for profile API responses in tests
type ProfileResponse struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Email              string   `json:"email"`
	Username           string   `json:"username"`
	DietaryPreferences []string `json:"dietary_preferences"`
	Allergies          []string `json:"allergies"`
}

// UpdateProfileRequest is used for updating a profile in tests
type UpdateProfileRequest struct {
	Name               string   `json:"name"`
	Username           string   `json:"username"`
	DietaryPreferences []string `json:"dietary_preferences"`
	Allergies          []string `json:"allergies"`
}
