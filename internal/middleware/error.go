package middleware

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// responseRecorder is a custom ResponseWriter to capture status and body
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       string
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.statusCode >= 400 {
		r.body = strings.TrimSpace(string(b))
		// Do not write the original error body to the response
		return len(b), nil
	}
	return r.ResponseWriter.Write(b)
}

// ErrorHandler is a middleware that logs errors and returns a JSON error response
func ErrorHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w, statusCode: 200}
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Error: %v", err)
				rec.Header().Set("Content-Type", "application/json")
				rec.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(rec).Encode(ErrorResponse{Error: "Internal Server Error"})
			} else if rec.statusCode >= 400 {
				// If an error status was written, return JSON error
				rec.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(ErrorResponse{Error: rec.body})
			}
		}()

		next.ServeHTTP(rec, r)
	})
}
