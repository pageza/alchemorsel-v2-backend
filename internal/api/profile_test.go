package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetProfileHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/user/profile", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetProfileHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response Profile
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	expected := Profile{
		ID:       "1",
		Username: "testuser",
		Email:    "test@example.com",
	}

	if response != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", response, expected)
	}
}

func TestUpdateProfileHandler(t *testing.T) {
	profile := Profile{
		ID:       "1",
		Username: "updateduser",
		Email:    "updated@example.com",
	}
	jsonData, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PUT", "/api/user/profile", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(UpdateProfileHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if response["status"] != "success" {
		t.Errorf("handler returned unexpected body: got %v want %v", response, map[string]string{"status": "success"})
	}
}
