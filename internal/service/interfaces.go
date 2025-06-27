package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
)

type LLMServiceInterface interface {
	// Legacy methods (maintain backward compatibility)
	GenerateRecipe(query string, dietaryPrefs []string, allergens []string, draft *RecipeDraft) (string, error)
	SaveDraft(ctx context.Context, draft *RecipeDraft) error
	GetDraft(ctx context.Context, draftID string) (*RecipeDraft, error)
	UpdateDraft(ctx context.Context, draft *RecipeDraft) error
	DeleteDraft(ctx context.Context, id string) error
	CalculateMacros(ingredients []string) (*Macros, error)
	GenerateRecipesBatch(prompts []string) ([]string, error)
	
	// Multi-call recipe generation methods
	GenerateBasicRecipe(ctx context.Context, query string, dietaryPrefs []string, allergens []string, userID string) (*RecipeDraft, error)
	CalculateRecipeNutrition(ctx context.Context, draftID string) (*Macros, error)
	FinalizeRecipe(ctx context.Context, draftID string) (*RecipeDraft, error)
}

// IAuthService defines the interface for authentication operations
type IAuthService interface {
	Register(ctx context.Context, email, password, username string, prefs *types.UserPreferences) (*models.User, error)
	Login(ctx context.Context, email, password string) (*models.User, *models.UserProfile, error)
	ValidateToken(token string) (*types.TokenClaims, error)
	GenerateToken(claims *types.TokenClaims) (string, error)
	GenerateVerificationToken(ctx context.Context, userID uuid.UUID) (string, error)
	ValidateVerificationToken(ctx context.Context, token string) (*models.User, error)
	ResendVerificationEmail(ctx context.Context, email string, emailService IEmailService) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
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
	FavoriteRecipe(ctx context.Context, userID, recipeID uuid.UUID) error
	UnfavoriteRecipe(ctx context.Context, userID, recipeID uuid.UUID) error
	GetFavoriteRecipes(ctx context.Context, userID uuid.UUID) ([]*models.Recipe, error)
}

// IFeedbackService defines the interface for feedback operations
type IFeedbackService interface {
	CreateFeedback(ctx context.Context, req *types.CreateFeedbackRequest, userID *uuid.UUID) (*models.Feedback, error)
	GetFeedback(ctx context.Context, id uuid.UUID) (*models.Feedback, error)
	ListFeedback(ctx context.Context, filters *models.FeedbackFilters) ([]*models.Feedback, error)
	UpdateFeedbackStatus(ctx context.Context, id uuid.UUID, status string, adminNotes string) error
}

// IEmailService defines the interface for email operations
type IEmailService interface {
	SendFeedbackNotification(feedback *models.Feedback, user *models.User) error
	SendEmail(to, subject, body string) error
	SendVerificationEmail(user *models.User, token string) error
	SendWelcomeEmail(user *models.User) error
}
