package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorHandler(t *testing.T) {
	// Create a test handler that returns an error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Test error", http.StatusInternalServerError)
	})

	// Wrap the handler with the error handling middleware
	wrappedHandler := ErrorHandler(handler)

	// Create a test request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()

	// Call the wrapped handler
	wrappedHandler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
	}

	// Check the response body
	expected := `{"error":"Test error"}` + "\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
