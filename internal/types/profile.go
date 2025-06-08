package types

import (
	"time"

	"github.com/google/uuid"
)

// UserProfile represents a user's profile
type UserProfile struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"user_id"`
	Username          string    `json:"username"`
	Bio               string    `json:"bio"`
	ProfilePictureURL string    `json:"profile_picture_url"`
	PrivacyLevel      string    `json:"privacy_level"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// UserPreferences represents a user's preferences
type UserPreferences struct {
	DietaryPrefs    []string `json:"dietary_prefs"`
	Allergies       []string `json:"allergies"`
	FavoriteCuisine string   `json:"favorite_cuisine"`
}

// UpdateProfileRequest represents a request to update a user's profile
type UpdateProfileRequest struct {
	Username          string           `json:"username,omitempty"`
	Bio               *string          `json:"bio,omitempty"`
	ProfilePictureURL *string          `json:"profile_picture_url,omitempty"`
	PrivacyLevel      *string          `json:"privacy_level,omitempty"`
	AvatarURL         string           `json:"avatar_url,omitempty"`
	Preferences       *UserPreferences `json:"preferences,omitempty"`
}

// ProfileHistory represents a user's profile history
type ProfileHistory struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedAt time.Time `json:"changed_at"`
	ChangedBy string    `json:"changed_by"`
}
