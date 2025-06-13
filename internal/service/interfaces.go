package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
)

type LLMServiceInterface interface {
	GenerateRecipe(query string, dietaryPrefs []string, allergens []string, draft *RecipeDraft) (string, error)
	SaveDraft(ctx context.Context, draft *RecipeDraft) error
	GetDraft(ctx context.Context, draftID string) (*RecipeDraft, error)
	UpdateDraft(ctx context.Context, draft *RecipeDraft) error
	DeleteDraft(ctx context.Context, id string) error
	CalculateMacros(ingredients []string) (*Macros, error)
	GenerateRecipesBatch(prompts []string) ([]string, error)
}

// IAuthService defines the interface for authentication operations
type IAuthService interface {
	Register(ctx context.Context, email, password string, prefs *types.UserPreferences) (*models.User, error)
	Login(ctx context.Context, email, password string) (*models.User, *models.UserProfile, error)
	ValidateToken(token string) (*types.TokenClaims, error)
	GenerateToken(claims *types.TokenClaims) (string, error)
}

// IProfileService defines the interface for user profile operations
type IProfileService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req *types.UpdateProfileRequest) (*models.UserProfile, error)
	Logout(ctx context.Context, userID uuid.UUID) error
	GetUserRecipes(ctx context.Context, userID uuid.UUID) ([]*models.Recipe, error)
	GetProfileHistory(ctx context.Context, userID uuid.UUID) ([]*types.ProfileHistory, error)
	GetUserProfile(username string) (*models.UserProfile, error)
}

// IRecipeService defines the interface for recipe operations
type IRecipeService interface {
	CreateRecipe(ctx context.Context, recipe *models.Recipe) (*models.Recipe, error)
	GetRecipe(ctx context.Context, id uuid.UUID) (*models.Recipe, error)
	UpdateRecipe(ctx context.Context, id uuid.UUID, recipe *models.Recipe) (*models.Recipe, error)
	DeleteRecipe(ctx context.Context, id uuid.UUID) error
	ListRecipes(ctx context.Context, userID *uuid.UUID) ([]*models.Recipe, error)
	SearchRecipes(ctx context.Context, query string) ([]*models.Recipe, error)
}
