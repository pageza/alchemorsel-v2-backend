package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pageza/alchemorsel-v2/backend/internal/model"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRecipeTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	// Manually create tables using simplified types since the production
	// models rely on PostgreSQL features not supported by SQLite.
	createRecipes := `CREATE TABLE recipes (
               id TEXT PRIMARY KEY,
               created_at DATETIME,
               updated_at DATETIME,
               deleted_at DATETIME,
               name TEXT,
               description TEXT,
               category TEXT,
               image_url TEXT,
               ingredients TEXT,
               instructions TEXT,
               calories REAL,
               protein REAL,
               carbs REAL,
               fat REAL,
               embedding TEXT,
               user_id TEXT
       );`
	if err := db.Exec(createRecipes).Error; err != nil {
		t.Fatalf("failed to create recipes table: %v", err)
	}

	createFavs := `CREATE TABLE recipe_favorites (
               id TEXT PRIMARY KEY,
               created_at DATETIME,
               updated_at DATETIME,
               recipe_id TEXT NOT NULL,
               user_id TEXT NOT NULL
       );`
	if err := db.Exec(createFavs).Error; err != nil {
		t.Fatalf("failed to create recipe_favorites table: %v", err)
	}
	return db
}

func TestFavoriteRecipe(t *testing.T) {
	db := setupRecipeTestDB(t)
	handler := NewRecipeHandler(db, nil)

	recipe := model.Recipe{
		ID:           uuid.New(),
		Name:         "Test",
		Embedding:    service.GenerateEmbedding("Test"),
		Ingredients:  model.JSONBStringArray{},
		Instructions: model.JSONBStringArray{},
		UserID:       uuid.New(),
	}
	db.Create(&recipe)

	userID := uuid.New()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: recipe.ID.String()}}
	c.Set("user_id", userID)

	handler.FavoriteRecipe(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, w.Code)
	}

	var count int64
	db.Model(&model.RecipeFavorite{}).Where("recipe_id = ? AND user_id = ?", recipe.ID, userID).Count(&count)
	if count != 1 {
		t.Fatalf("favorite not created")
	}
}

func TestUnfavoriteRecipe(t *testing.T) {
	db := setupRecipeTestDB(t)
	handler := NewRecipeHandler(db, nil)

	recipe := model.Recipe{
		ID:           uuid.New(),
		Name:         "Test",
		Embedding:    service.GenerateEmbedding("Test"),
		Ingredients:  model.JSONBStringArray{},
		Instructions: model.JSONBStringArray{},
		UserID:       uuid.New(),
	}
	db.Create(&recipe)
	userID := uuid.New()
	fav := model.RecipeFavorite{RecipeID: recipe.ID, UserID: userID}
	db.Create(&fav)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = []gin.Param{{Key: "id", Value: recipe.ID.String()}}
	c.Set("user_id", userID)

	handler.UnfavoriteRecipe(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, w.Code)
	}

	var count int64
	db.Model(&model.RecipeFavorite{}).Where("recipe_id = ? AND user_id = ?", recipe.ID, userID).Count(&count)
	if count != 0 {
		t.Fatalf("favorite not removed")
	}
}

func TestListRecipesFilters(t *testing.T) {
	db := setupRecipeTestDB(t)
	handler := NewRecipeHandler(db, nil)

	r1 := model.Recipe{
		ID:           uuid.New(),
		Name:         "Pasta",
		Description:  "Tasty pasta dish",
		Category:     "Italian",
		Ingredients:  model.JSONBStringArray{},
		Instructions: model.JSONBStringArray{},
		Embedding:    service.GenerateEmbedding("Pasta Tasty pasta dish"),
		UserID:       uuid.New(),
	}
	r2 := model.Recipe{
		ID:           uuid.New(),
		Name:         "Salad",
		Description:  "Healthy salad",
		Category:     "Healthy",
		Ingredients:  model.JSONBStringArray{},
		Instructions: model.JSONBStringArray{},
		Embedding:    service.GenerateEmbedding("Salad Healthy salad"),
		UserID:       uuid.New(),
	}
	r3 := model.Recipe{
		ID:           uuid.New(),
		Name:         "Pasta Carbonara",
		Description:  "Creamy",
		Category:     "Italian",
		Ingredients:  model.JSONBStringArray{},
		Instructions: model.JSONBStringArray{},
		Embedding:    service.GenerateEmbedding("Pasta Carbonara Creamy"),
		UserID:       uuid.New(),
	}
	db.Create(&r1)
	db.Create(&r2)
	db.Create(&r3)

	// search by q
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/recipes?q=pasta", nil)
	handler.ListRecipes(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, w.Code)
	}
	var resp struct {
		Recipes []model.Recipe `json:"recipes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Recipes) != 2 {
		t.Fatalf("expected 2 recipes got %d", len(resp.Recipes))
	}

	// filter by category
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/recipes?category=Healthy", nil)
	handler.ListRecipes(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, w.Code)
	}
	resp = struct {
		Recipes []model.Recipe `json:"recipes"`
	}{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Recipes) != 1 {
		t.Fatalf("expected 1 recipe got %d", len(resp.Recipes))
	}
}
