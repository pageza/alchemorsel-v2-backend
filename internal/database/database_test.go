package database

import (
	"os"
	"testing"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Setup
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "alchemorsel")
	os.Setenv("DB_PASSWORD", "postgres")
	os.Setenv("DB_NAME", "alchemorsel_test")
	os.Setenv("DB_SSL_MODE", "disable")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

func TestDatabaseConnection(t *testing.T) {
	cfg := &config.Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
	}

	db, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, db)

	// Test connection
	err = db.Ping()
	assert.NoError(t, err)
}
