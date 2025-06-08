package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	pgvector "github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// JSONBStringArray is a custom type for handling string arrays in JSONB
type JSONBStringArray []string

// Value implements the driver.Valuer interface
func (a JSONBStringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface
func (a *JSONBStringArray) Scan(value interface{}) error {
	if value == nil {
		*a = JSONBStringArray{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, a)
}

type Recipe struct {
	ID           uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	DeletedAt    gorm.DeletedAt   `gorm:"index" json:"-"`
	Name         string           `gorm:"size:255;not null" json:"name"`
	Description  string           `gorm:"type:text" json:"description"`
	Category     string           `gorm:"size:50" json:"category"`
	ImageURL     string           `gorm:"size:255" json:"image_url"`
	Ingredients  JSONBStringArray `gorm:"type:jsonb;not null;default:'[]'" json:"ingredients"`
	Instructions JSONBStringArray `gorm:"type:jsonb;not null;default:'[]'" json:"instructions"`
	Calories     float64          `gorm:"type:float" json:"calories"`
	Protein      float64          `gorm:"type:float" json:"protein"`
	Carbs        float64          `gorm:"type:float" json:"carbs"`
	Fat          float64          `gorm:"type:float" json:"fat"`
	Embedding    pgvector.Vector  `gorm:"type:vector(1536)" json:"-"`
	UserID       uuid.UUID        `gorm:"type:uuid;not null" json:"user_id"`
}
