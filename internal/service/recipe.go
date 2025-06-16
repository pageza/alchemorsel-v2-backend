package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/models"
	"gorm.io/gorm"
)

// RecipeService handles recipe operations
type RecipeService struct {
	db               *gorm.DB
	embeddingService EmbeddingServiceInterface
}

// NewRecipeService creates a new RecipeService instance
func NewRecipeService(db *gorm.DB, embeddingService EmbeddingServiceInterface) *RecipeService {
	return &RecipeService{
		db:               db,
		embeddingService: embeddingService,
	}
}

// CreateRecipe creates a new recipe
func (s *RecipeService) CreateRecipe(ctx context.Context, recipe *models.Recipe) (*models.Recipe, error) {
	if err := s.db.Create(recipe).Error; err != nil {
		return nil, err
	}
	return recipe, nil
}

// GetRecipe retrieves a recipe by ID
func (s *RecipeService) GetRecipe(ctx context.Context, id uuid.UUID) (*models.Recipe, error) {
	var recipe models.Recipe
	if err := s.db.First(&recipe, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &recipe, nil
}

// UpdateRecipe updates a recipe
func (s *RecipeService) UpdateRecipe(ctx context.Context, id uuid.UUID, recipe *models.Recipe) (*models.Recipe, error) {
	if err := s.db.Model(&models.Recipe{}).Where("id = ?", id).Updates(recipe).Error; err != nil {
		return nil, err
	}
	return s.GetRecipe(ctx, id)
}

// DeleteRecipe deletes a recipe
func (s *RecipeService) DeleteRecipe(ctx context.Context, id uuid.UUID) error {
	// First check if the recipe exists
	var recipe models.Recipe
	if err := s.db.First(&recipe, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return gorm.ErrRecordNotFound
		}
		return err
	}

	// If recipe exists, delete it
	return s.db.Delete(&models.Recipe{}, "id = ?", id).Error
}

// ListRecipes lists recipes for a user or all users if userID is nil
func (s *RecipeService) ListRecipes(ctx context.Context, userID *uuid.UUID) ([]*models.Recipe, error) {
	var recipes []models.Recipe
	query := s.db
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if err := query.Find(&recipes).Error; err != nil {
		return nil, err
	}
	// Convert to []*models.Recipe
	result := make([]*models.Recipe, len(recipes))
	for i := range recipes {
		result[i] = &recipes[i]
	}
	return result, nil
}

// SearchRecipes searches for recipes
func (s *RecipeService) SearchRecipes(ctx context.Context, query string) ([]*models.Recipe, error) {
	var recipes []models.Recipe

	dbQuery := s.db

	if query != "" {
		if s.db.Dialector.Name() == "postgres" {
			// Generate embedding for semantic search
			vec, err := s.embeddingService.GenerateEmbedding(query)
			if err != nil {
				return nil, err
			}

			// Combine semantic and keyword search
			// Use a subquery to get both semantic and keyword matches
			subQuery := s.db.Model(&models.Recipe{}).
				Select("id, embedding <-> ? as similarity", vec).
				Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(ingredients::text) LIKE ?",
					"%"+strings.ToLower(query)+"%",
					"%"+strings.ToLower(query)+"%",
					"%"+strings.ToLower(query)+"%",
				)

			// Join with the main query and order by similarity
			dbQuery = dbQuery.Joins("JOIN (?) as search ON recipes.id = search.id", subQuery).
				Order("search.similarity ASC")
		} else {
			// Fallback to keyword search for non-PostgreSQL databases
			like := "%" + strings.ToLower(query) + "%"
			dbQuery = dbQuery.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(ingredients) LIKE ?",
				like, like, like)
		}
	}

	if err := dbQuery.Find(&recipes).Error; err != nil {
		return nil, err
	}

	// Convert to []*models.Recipe
	result := make([]*models.Recipe, len(recipes))
	for i := range recipes {
		result[i] = &recipes[i]
	}
	return result, nil
}

// GetRecipeWithFavoriteStatus retrieves a recipe by ID and returns its favorite status for the user
func (s *RecipeService) GetRecipeWithFavoriteStatus(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*models.Recipe, bool, error) {
	recipe, err := s.GetRecipe(ctx, id)
	if err != nil {
		return nil, false, err
	}

	isFavorite, err := s.IsRecipeFavorited(ctx, userID, id)
	if err != nil {
		return recipe, false, err
	}

	return recipe, isFavorite, nil
}

// FavoriteRecipe adds a recipe to user's favorites
func (s *RecipeService) FavoriteRecipe(ctx context.Context, userID uuid.UUID, recipeID uuid.UUID) error {
	// Check if recipe exists
	var recipe models.Recipe
	if err := s.db.First(&recipe, "id = ?", recipeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return gorm.ErrRecordNotFound
		}
		return err
	}

	// Check if already favorited (excluding soft-deleted records)
	var existing models.RecipeFavorite
	if err := s.db.Where("user_id = ? AND recipe_id = ?", userID, recipeID).First(&existing).Error; err == nil {
		// Already favorited
		return gorm.ErrDuplicatedKey
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	// Check if there's a soft-deleted favorite we can restore
	var softDeleted models.RecipeFavorite
	if err := s.db.Unscoped().Where("user_id = ? AND recipe_id = ? AND deleted_at IS NOT NULL", userID, recipeID).First(&softDeleted).Error; err == nil {
		// Restore the soft-deleted favorite
		return s.db.Unscoped().Model(&softDeleted).Update("deleted_at", nil).Error
	}

	// Create new favorite
	favorite := models.RecipeFavorite{
		UserID:   userID,
		RecipeID: recipeID,
	}

	return s.db.Create(&favorite).Error
}

// UnfavoriteRecipe removes a recipe from user's favorites
func (s *RecipeService) UnfavoriteRecipe(ctx context.Context, userID uuid.UUID, recipeID uuid.UUID) error {
	// Check if favorite exists
	var favorite models.RecipeFavorite
	if err := s.db.Where("user_id = ? AND recipe_id = ?", userID, recipeID).First(&favorite).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return gorm.ErrRecordNotFound
		}
		return err
	}

	// Delete the favorite
	return s.db.Delete(&favorite).Error
}

// IsRecipeFavorited checks if a recipe is favorited by the user
func (s *RecipeService) IsRecipeFavorited(ctx context.Context, userID uuid.UUID, recipeID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&models.RecipeFavorite{}).
		Where("user_id = ? AND recipe_id = ?", userID, recipeID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
