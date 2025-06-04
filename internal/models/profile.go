package models

import (
	"time"

	"gorm.io/gorm"
)

type Profile struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	UserID    uint           `gorm:"not null;uniqueIndex" json:"user_id"`
	Username  string         `gorm:"size:255;not null" json:"username"`
	Email     string         `gorm:"size:255;not null;uniqueIndex" json:"email"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
