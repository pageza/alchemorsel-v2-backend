package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                           uuid.UUID           `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	CreatedAt                    time.Time           `json:"created_at"`
	UpdatedAt                    time.Time           `json:"updated_at"`
	DeletedAt                    gorm.DeletedAt      `gorm:"index" json:"-"`
	Name                         string              `gorm:"not null" json:"name"`
	Email                        string              `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash                 string              `gorm:"not null" json:"-"`
	EmailVerified                bool                `gorm:"column:is_email_verified;default:false" json:"is_email_verified"`
	EmailVerifiedAt              *time.Time          `gorm:"column:email_verified_at" json:"email_verified_at,omitempty"`
	VerificationToken            *string             `gorm:"column:verification_token" json:"-"`
	VerificationTokenExpiresAt   *time.Time          `gorm:"column:verification_token_expires_at" json:"-"`
	Profile                      UserProfile         `gorm:"foreignKey:UserID" json:"profile"`
	DietaryPrefs                 []DietaryPreference `gorm:"foreignKey:UserID" json:"dietary_preferences"`
	Allergens                    []Allergen          `gorm:"foreignKey:UserID" json:"allergens"`
}

type UserProfile struct {
	ID                uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID            uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Username          string         `gorm:"size:50;not null;uniqueIndex" json:"username"`
	Bio               string         `gorm:"type:text" json:"bio"`
	ProfilePictureURL string         `gorm:"size:255" json:"profile_picture_url"`
	PrivacyLevel      string         `gorm:"size:50;not null;default:'private'" json:"privacy_level"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}
