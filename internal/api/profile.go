package api

import (
	"encoding/json"
	"net/http"
)

// Profile represents a user profile
type Profile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// GetProfileHandler handles GET /api/user/profile
func GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	// Mock profile response
	profile := Profile{
		ID:       "1",
		Username: "testuser",
		Email:    "test@example.com",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// UpdateProfileHandler handles PUT /api/user/profile
func UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	var profile Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Mock update success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// RegisterProfileRoutes registers the profile API routes
func RegisterProfileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/user/profile", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetProfileHandler(w, r)
		case http.MethodPut:
			UpdateProfileHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
