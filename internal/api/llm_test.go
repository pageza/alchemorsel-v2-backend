package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/model"
	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

func setupLLMDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
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
	return db
}

func TestQuerySavesRecipe(t *testing.T) {
	db := setupLLMDB(t)

	t.Setenv("DEEPSEEK_API_KEY", "dummy")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"name\":\"Mock Recipe\",\"description\":\"Desc\",\"category\":\"Cat\",\"ingredients\":[\"i1\"],\"instructions\":[\"s1\"]}"}}]}`)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_URL", ts.URL)
	authSvc := service.NewAuthService(nil, "secret")
	handler, err := NewLLMHandler(db, authSvc)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	// create token
	userID := uuid.New()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	body := `{"query":"test","intent":"generate"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/llm/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d got %d", http.StatusCreated, w.Code)
	}

	var resp struct {
		Recipe model.Recipe `json:"recipe"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Recipe.ID == uuid.Nil {
		t.Fatalf("recipe ID not set")
	}
	if resp.Recipe.UserID != userID {
		t.Fatalf("user id mismatch")
	}

	var record model.Recipe
	if err := db.First(&record, "id = ?", resp.Recipe.ID.String()).Error; err != nil {
		t.Fatalf("recipe not saved")
	}
	if record.UserID != userID {
		t.Fatalf("user id not persisted")
	}
}

func TestQueryUnauthorized(t *testing.T) {
	db := setupLLMDB(t)

	t.Setenv("DEEPSEEK_API_KEY", "dummy")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{}"}}]}`)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_URL", ts.URL)
	authSvc := service.NewAuthService(nil, "secret")
	handler, err := NewLLMHandler(db, authSvc)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	body := `{"query":"test","intent":"generate"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/llm/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestQueryModifyRecipe(t *testing.T) {
	db := setupLLMDB(t)

	t.Setenv("DEEPSEEK_API_KEY", "dummy")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"name\":\"Updated Recipe\",\"description\":\"New Desc\",\"category\":\"NewCat\",\"ingredients\":[\"i2\"],\"instructions\":[\"s2\"]}"}}]}`)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_URL", ts.URL)
	authSvc := service.NewAuthService(nil, "secret")
	handler, err := NewLLMHandler(db, authSvc)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	userID := uuid.New()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	orig := model.Recipe{
		ID:           uuid.New(),
		Name:         "Old",
		Description:  "Old",
		Category:     "Cat",
		Ingredients:  model.JSONBStringArray{"x"},
		Instructions: model.JSONBStringArray{"y"},
		UserID:       userID,
		Embedding:    service.GenerateEmbedding("Old Old"),
	}
	if err := db.Create(&orig).Error; err != nil {
		t.Fatalf("failed to seed recipe: %v", err)
	}

	body := fmt.Sprintf(`{"query":"change it","intent":"modify","recipe_id":"%s"}`, orig.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/llm/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, w.Code)
	}

	var resp struct {
		Recipe model.Recipe `json:"recipe"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Recipe.ID != orig.ID {
		t.Fatalf("recipe id changed")
	}
	if resp.Recipe.Name != "Updated Recipe" {
		t.Fatalf("name not updated")
	}

	var record model.Recipe
	if err := db.First(&record, "id = ?", orig.ID.String()).Error; err != nil {
		t.Fatalf("recipe not saved")
	}
	if record.Name != "Updated Recipe" {
		t.Fatalf("db not updated")
	}
}

func TestQueryIncludesPreferences(t *testing.T) {
	db := setupLLMDB(t)

	var captured string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req service.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if captured == "" {
			for _, m := range req.Messages {
				if m.Role == "user" {
					captured = m.Content
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{}"}}]}`)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_KEY", "dummy")
	t.Setenv("DEEPSEEK_API_URL", ts.URL)
	authSvc := service.NewAuthService(nil, "secret")
	handler, err := NewLLMHandler(db, authSvc)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	userID := uuid.New()
	db.Exec("INSERT INTO dietary_preferences (id, user_id, preference_type) VALUES (?, ?, ?)", uuid.New().String(), userID.String(), "vegan")
	db.Exec("INSERT INTO allergens (id, user_id, allergen_name, severity_level) VALUES (?, ?, ?, 1)", uuid.New().String(), userID.String(), "peanuts")

	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte("secret"))

	body := `{"query":"test","intent":"generate"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/llm/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !strings.Contains(captured, "vegan") || !strings.Contains(captured, "peanuts") {
		t.Fatalf("prompt missing preferences: %s", captured)
	}
}
