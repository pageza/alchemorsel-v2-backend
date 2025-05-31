package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pageza/alchemorsel-v2/backend/config"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Setup
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	os.Setenv("DB_NAME", "alchemorsel_test")
	os.Setenv("DB_SSL_MODE", "disable")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

func TestNew(t *testing.T) {
	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBPassword: "postgres",
		DBName:     "alchemorsel_test",
		DBSSLMode:  "disable",
	}

	db, err := New(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.NotNil(t, db.GormDB)

	// Test health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.HealthCheck(ctx)
	assert.NoError(t, err)

	// Test closing the connection
	err = db.Close()
	assert.NoError(t, err)
}

func TestNewWithInvalidConfig(t *testing.T) {
	cfg := &config.Config{
		DBHost:     "invalid-host",
		DBPort:     "5432",
		DBUser:     "postgres",
		DBPassword: "postgres",
		DBName:     "alchemorsel_test",
		DBSSLMode:  "disable",
	}

	db, err := New(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
}
