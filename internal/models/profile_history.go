package models

import (
	"time"

	"gorm.io/gorm"
)

// ProfileHistory represents a record of profile changes
type ProfileHistory struct {
	gorm.Model
	UserID    string    `gorm:"index;not null"`
	Field     string    `gorm:"not null"` // The field that was changed
	OldValue  string    `gorm:"type:text"`
	NewValue  string    `gorm:"type:text"`
	ChangedAt time.Time `gorm:"not null"`
	ChangedBy string    `gorm:"not null"` // User ID of who made the change
}

// TableName specifies the table name for ProfileHistory
func (ProfileHistory) TableName() string {
	return "profile_history"
}
