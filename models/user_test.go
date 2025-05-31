package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	db.AutoMigrate(&User{})
	return db
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	user := &User{Username: "testuser", Email: "test@example.com"}
	err := CreateUser(db, user)
	if err != nil {
		t.Errorf("Failed to create user: %v", err)
	}
	if user.ID == 0 {
		t.Error("User ID should be set after creation")
	}
}

func TestGetUser(t *testing.T) {
	db := setupTestDB(t)
	user := &User{Username: "testuser", Email: "test@example.com"}
	CreateUser(db, user)
	retrievedUser, err := GetUser(db, user.ID)
	if err != nil {
		t.Errorf("Failed to get user: %v", err)
	}
	if retrievedUser.Username != user.Username {
		t.Error("Retrieved user username does not match")
	}
}

func TestUpdateUser(t *testing.T) {
	db := setupTestDB(t)
	user := &User{Username: "testuser", Email: "test@example.com"}
	CreateUser(db, user)
	user.Username = "updateduser"
	err := UpdateUser(db, user)
	if err != nil {
		t.Errorf("Failed to update user: %v", err)
	}
	retrievedUser, _ := GetUser(db, user.ID)
	if retrievedUser.Username != "updateduser" {
		t.Error("User username was not updated")
	}
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	user := &User{Username: "testuser", Email: "test@example.com"}
	CreateUser(db, user)
	err := DeleteUser(db, user.ID)
	if err != nil {
		t.Errorf("Failed to delete user: %v", err)
	}
	_, err = GetUser(db, user.ID)
	if err == nil {
		t.Error("User should be deleted")
	}
}
