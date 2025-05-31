package config

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")

	// Create a new config
	cfg := New()

	// Verify that the config values match the environment variables
	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort = %v; want %v", cfg.ServerPort, "8080")
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("DBHost = %v; want %v", cfg.DBHost, "localhost")
	}
	if cfg.DBPort != "5432" {
		t.Errorf("DBPort = %v; want %v", cfg.DBPort, "5432")
	}
	if cfg.DBUser != "testuser" {
		t.Errorf("DBUser = %v; want %v", cfg.DBUser, "testuser")
	}
	if cfg.DBPassword != "testpass" {
		t.Errorf("DBPassword = %v; want %v", cfg.DBPassword, "testpass")
	}
	if cfg.DBName != "testdb" {
		t.Errorf("DBName = %v; want %v", cfg.DBName, "testdb")
	}

	// Clear environment variables to test default values
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")

	// Create a new config with default values
	cfg = New()

	// Verify that the config values match the default values
	if cfg.ServerPort != "8080" {
		t.Errorf("ServerPort = %v; want %v", cfg.ServerPort, "8080")
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("DBHost = %v; want %v", cfg.DBHost, "localhost")
	}
	if cfg.DBPort != "5432" {
		t.Errorf("DBPort = %v; want %v", cfg.DBPort, "5432")
	}
	if cfg.DBUser != "" {
		t.Errorf("DBUser = %v; want %v", cfg.DBUser, "")
	}
	if cfg.DBPassword != "" {
		t.Errorf("DBPassword = %v; want %v", cfg.DBPassword, "")
	}
	if cfg.DBName != "" {
		t.Errorf("DBName = %v; want %v", cfg.DBName, "")
	}
}
