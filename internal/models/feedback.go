package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Feedback struct {
	ID          uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	UserID      *uuid.UUID     `gorm:"type:uuid" json:"user_id,omitempty"`
	User        *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Type        string         `gorm:"not null" json:"type"` // bug, feature, general
	Title       string         `gorm:"not null" json:"title"`
	Description string         `gorm:"type:text;not null" json:"description"`
	Priority    string         `gorm:"default:'medium'" json:"priority"` // low, medium, high, critical
	Status      string         `gorm:"default:'open'" json:"status"`     // open, in_progress, resolved, closed
	UserAgent   string         `json:"user_agent"`
	URL         string         `json:"url"`
	AdminNotes  string         `gorm:"type:text" json:"admin_notes"`
}

// TableName returns the table name for the Feedback model
func (Feedback) TableName() string {
	return "feedback"
}

// FeedbackFilters represents filters for listing feedback
type FeedbackFilters struct {
	Type     string `json:"type,omitempty"`
	Status   string `json:"status,omitempty"`
	Priority string `json:"priority,omitempty"`
	UserID   string `json:"user_id,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}
