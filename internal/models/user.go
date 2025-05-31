package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:varchar(36);primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"not null" json:"name"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
}

type UserProfile struct {
	ID                   uuid.UUID      `gorm:"type:varchar(36);primarykey" json:"id"`
	UserID               uuid.UUID      `gorm:"type:varchar(36);not null;uniqueIndex" json:"user_id"`
	Username             string         `gorm:"size:50;not null;uniqueIndex" json:"username"`
	Bio                  string         `gorm:"type:text" json:"bio"`
	ProfilePictureURL    string         `gorm:"size:255" json:"profile_picture_url"`
	PrivacyLevel         string         `gorm:"not null;default:'private'" json:"privacy_level"`
	CookingAbilityLevel  string         `gorm:"default:'beginner'" json:"cooking_ability_level"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

type DietaryPreference struct {
	ID             uuid.UUID      `gorm:"type:varchar(36);primarykey" json:"id"`
	UserID         uuid.UUID      `gorm:"type:varchar(36);not null" json:"user_id"`
	PreferenceType string         `gorm:"not null" json:"preference_type"`
	CustomName     string         `gorm:"size:50" json:"custom_name"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Allergen struct {
	ID            uuid.UUID `gorm:"type:varchar(36);primarykey" json:"id"`
	UserID        uuid.UUID `gorm:"type:varchar(36);not null" json:"user_id"`
	AllergenName  string    `gorm:"size:50;not null" json:"allergen_name"`
	SeverityLevel int       `gorm:"not null;check:severity_level >= 1 AND severity_level <= 5" json:"severity_level"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type UserAppliance struct {
	ID            uuid.UUID      `gorm:"type:varchar(36);primarykey" json:"id"`
	UserID        uuid.UUID      `gorm:"type:varchar(36);not null" json:"user_id"`
	ApplianceType string         `gorm:"not null" json:"appliance_type"`
	CustomName    string         `gorm:"size:50" json:"custom_name"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
