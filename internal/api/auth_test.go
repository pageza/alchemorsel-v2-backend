package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/pageza/alchemorsel-v2/backend/internal/service"
)

func setupAuthDB(t *testing.T) *gorm.DB {
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
	createAllergens := `CREATE TABLE allergens (
        id TEXT PRIMARY KEY,
        created_at DATETIME,
        updated_at DATETIME,
        deleted_at DATETIME,
        user_id TEXT,
        allergen_name TEXT,
        severity_level INTEGER
    );`
	if err := db.Exec(createAllergens).Error; err != nil {
		t.Fatalf("failed to create allergens table: %v", err)
	}
	return db
}

func TestRegisterMissingPrefs(t *testing.T) {
	db := setupAuthDB(t)
	svc := service.NewAuthService(db, "secret")
	h := NewAuthHandler(svc)
	router := gin.New()
	v1 := router.Group("/api/v1")
	h.RegisterRoutes(v1)

	body := `{"name":"Tester","email":"t@example.com","password":"password","username":"tester"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 got %d", w.Code)
	}
}

func TestRegisterWithPrefs(t *testing.T) {
	db := setupAuthDB(t)
	svc := service.NewAuthService(db, "secret")
	h := NewAuthHandler(svc)
	router := gin.New()
	v1 := router.Group("/api/v1")
	h.RegisterRoutes(v1)

	reqBody := map[string]interface{}{
		"name":                "Tester",
		"email":               "t2@example.com",
		"password":            "password",
		"username":            "tester2",
		"dietary_preferences": []string{"vegan"},
		"allergies":           []string{"peanuts"},
	}
	buf, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse resp: %v", err)
	}
	if resp["token"] == "" {
		t.Fatalf("token missing")
	}
}
