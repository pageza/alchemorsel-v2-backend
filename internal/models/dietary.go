package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DietaryPreference represents a user's dietary preference entry.
type DietaryPreference struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	PreferenceType string         `gorm:"type:dietary_preference_type;not null" json:"preference_type"`
	CustomName     string         `gorm:"size:50" json:"custom_name"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (DietaryPreference) TableName() string {
	return "dietary_preferences"
}

// Allergen represents an allergen entry for a user.
type Allergen struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	AllergenName  string         `gorm:"size:50;not null" json:"allergen_name"`
	SeverityLevel int            `gorm:"not null" json:"severity_level"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Allergen) TableName() string {
	return "allergens"
}
