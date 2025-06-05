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
           embedding TEXT,
           user_id TEXT
   );`
	if err := db.Exec(createRecipes).Error; err != nil {
		t.Fatalf("failed to create recipes table: %v", err)
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
