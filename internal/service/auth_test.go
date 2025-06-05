package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	svc := NewAuthService(db, "secret")
	_, err := svc.Register("Tester", "t@example.com", "password", "tester", "", "")
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != ErrMissingPreferences {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterWithPrefs(t *testing.T) {
	db := setupAuthDB(t)
	svc := NewAuthService(db, "secret")
	token, err := svc.Register("Tester", "t2@example.com", "password", "tester2", "vegan", "peanuts")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if token == "" {
		t.Fatalf("token not returned")
	}
}
