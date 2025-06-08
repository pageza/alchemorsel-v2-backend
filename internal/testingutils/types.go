package testingutils

import (
	"github.com/pageza/alchemorsel-v2/backend/internal/testhelpers/types"
)

// Re-export commonly used types from testhelpers/types
type (
	CreateRecipeRequest  = types.CreateRecipeRequest
	UpdateRecipeRequest  = types.UpdateRecipeRequest
	RecipeResponse       = types.RecipeResponse
	ListRecipesResponse  = types.ListRecipesResponse
	RegisterRequest      = types.RegisterRequest
	LoginRequest         = types.LoginRequest
	LoginResponse        = types.LoginResponse
	ProfileResponse      = types.ProfileResponse
	UpdateProfileRequest = types.UpdateProfileRequest
)
