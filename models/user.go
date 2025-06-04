package models

import (
	"gorm.io/gorm"
)

// User represents a user profile
type User struct {
	gorm.Model
	Username string `json:"username" gorm:"unique;not null;size:50"`
	Email    string `json:"email" gorm:"unique;not null;size:100"`
}

// CreateUser inserts a new user into the database
func CreateUser(db *gorm.DB, user *User) error {
	return db.Create(user).Error
}

// GetUser retrieves a user by ID
func GetUser(db *gorm.DB, id uint) (*User, error) {
	var user User
	err := db.First(&user, id).Error
	return &user, err
}

// UpdateUser updates a user's profile
func UpdateUser(db *gorm.DB, user *User) error {
	return db.Save(user).Error
}

// DeleteUser removes a user from the database
func DeleteUser(db *gorm.DB, id uint) error {
	return db.Delete(&User{}, id).Error
}
