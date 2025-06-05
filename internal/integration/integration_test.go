package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/api"
	"github.com/pageza/alchemorsel-v2/backend/internal/model"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	createUsers := `CREATE TABLE users (
        id TEXT PRIMARY KEY,
        created_at DATETIME,
        updated_at DATETIME,
        deleted_at DATETIME,
        name TEXT,
        email TEXT UNIQUE,
        password_hash TEXT
    );`
	if err := db.Exec(createUsers).Error; err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}
	createProfiles := `CREATE TABLE user_profiles (
        id TEXT PRIMARY KEY,
        created_at DATETIME,
        updated_at DATETIME,
        deleted_at DATETIME,
        user_id TEXT,
        username TEXT,
        email TEXT,
        bio TEXT,
        profile_picture_url TEXT,
        privacy_level TEXT
    );`
	if err := db.Exec(createProfiles).Error; err != nil {
		t.Fatalf("failed to create user_profiles table: %v", err)
	}
	createPrefs := `CREATE TABLE dietary_preferences (
        id TEXT PRIMARY KEY,
        created_at DATETIME,
        updated_at DATETIME,
        deleted_at DATETIME,
        user_id TEXT,
        preference_type TEXT,
        custom_name TEXT
    );`
	if err := db.Exec(createPrefs).Error; err != nil {
		t.Fatalf("failed to create dietary_preferences table: %v", err)
	}
	createAlls := `CREATE TABLE allergens (
        id TEXT PRIMARY KEY,
        created_at DATETIME,
        updated_at DATETIME,
        deleted_at DATETIME,
        user_id TEXT,
        allergen_name TEXT,
        severity_level INTEGER
    );`
	if err := db.Exec(createAlls).Error; err != nil {
		t.Fatalf("failed to create allergens table: %v", err)
	}
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

func setupRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	authSvc := service.NewAuthService(db, "secret")
	profileSvc := service.NewProfileService(db, "secret")
	profileHandler := api.NewProfileHandler(profileSvc)
	authHandler := api.NewAuthHandler(authSvc)
	recipeHandler := api.NewRecipeHandler(db, authSvc)
	llmHandler, _ := api.NewLLMHandler(db, authSvc)

	v1 := router.Group("/api/v1")
	profileHandler.RegisterRoutes(v1)
	authHandler.RegisterRoutes(v1)
	recipeHandler.RegisterRoutes(v1)
	llmHandler.RegisterRoutes(v1)
	return router
}

func TestIntegrationRegisterLoginCreateModify(t *testing.T) {
	db := setupDB(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"name\":\"Test Recipe\",\"description\":\"Desc\",\"category\":\"Cat\",\"ingredients\":[\"i1\"],\"instructions\":[\"s1\"],\"calories\":100,\"protein\":10,\"carbs\":20,\"fat\":5}"}}]}`)
	}))
	defer ts.Close()
	t.Setenv("DEEPSEEK_API_KEY", "dummy")
	t.Setenv("DEEPSEEK_API_URL", ts.URL)

	router := setupRouter(db)

	regBody := map[string]interface{}{
		"name":                "Tester",
		"email":               "test@example.com",
		"password":            "password",
		"username":            "tester",
		"dietary_preferences": []string{"vegan"},
		"allergies":           []string{"peanuts"},
	}
	buf, err := json.Marshal(regBody)
	if err != nil {
		t.Fatalf("failed to marshal register body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("register failed: %d", w.Code)
	}
	var regResp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &regResp); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}
	if regResp["token"] == "" {
		t.Fatalf("no token returned")
	}

	loginBody := `{"email":"test@example.com","password":"password"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d", w.Code)
	}
	var loginResp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	token := loginResp["token"]
	if token == "" {
		t.Fatalf("no token from login")
	}

	reqBody := `{"query":"make something","intent":"generate"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/llm/query", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create recipe failed: %d", w.Code)
	}
	var createResp struct{ Recipe model.Recipe }
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if createResp.Recipe.ID == uuid.Nil {
		t.Fatalf("recipe id missing")
	}
	recipeID := createResp.Recipe.ID.String()

	ts.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"name\":\"Updated\",\"description\":\"New\",\"category\":\"Cat\",\"ingredients\":[\"x\"],\"instructions\":[\"y\"],\"calories\":150,\"protein\":15,\"carbs\":25,\"fat\":6}"}}]}`)
	})

	modBody := fmt.Sprintf(`{"query":"update","intent":"modify","recipe_id":"%s"}`, recipeID)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/llm/query", bytes.NewBufferString(modBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("modify recipe failed: %d", w.Code)
	}
	var modResp struct{ Recipe model.Recipe }
	if err := json.Unmarshal(w.Body.Bytes(), &modResp); err != nil {
		t.Fatalf("failed to decode modify response: %v", err)
	}
	if modResp.Recipe.Name != "Updated" {
		t.Fatalf("recipe not updated")
	}

	var rec model.Recipe
	if err := db.First(&rec, "id = ?", recipeID).Error; err != nil {
		t.Fatalf("recipe not in db: %v", err)
	}
	if rec.Name != "Updated" {
		t.Fatalf("db not updated")
	}
}
