package model

import (
	"github.com/google/uuid"
	"time"
)

type RecipeFavorite struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	RecipeID  uuid.UUID `gorm:"type:uuid;not null;index" json:"recipe_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
}

func (RecipeFavorite) TableName() string {
	return "recipe_favorites"
}
