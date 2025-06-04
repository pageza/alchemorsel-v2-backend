package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/model"
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
           user_id TEXT
   );`
	if err := db.Exec(createRecipes).Error; err != nil {
		t.Fatalf("failed to create recipes table: %v", err)
	}
	return db
}

func TestQuerySavesRecipe(t *testing.T) {
	db := setupLLMDB(t)

	tmpFile, err := os.CreateTemp(t.TempDir(), "key")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmpFile.WriteString("dummy"); err != nil {
		t.Fatalf("failed to write key: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}
	t.Setenv("DEEPSEEK_API_KEY_FILE", tmpFile.Name())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"name\":\"Mock Recipe\",\"description\":\"Desc\",\"category\":\"Cat\",\"ingredients\":[\"i1\"],\"instructions\":[\"s1\"]}"}}]}`)
	}))
	defer ts.Close()

	t.Setenv("DEEPSEEK_API_URL", ts.URL)
	handler, err := NewLLMHandler(db)
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}

	body := `{"query":"test","intent":"generate"}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/llm/query", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Query(c)

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

	var count int64
	db.Model(&model.Recipe{}).Where("id = ?", resp.Recipe.ID.String()).Count(&count)
	if count != 1 {
		t.Fatalf("recipe not saved")
	}
}
