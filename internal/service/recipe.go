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
