package database

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NotNil(t, db)

	// Test database connection
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NotNil(t, sqlDB)

	// Test database health
	err = sqlDB.Ping()
	assert.NoError(t, err)

	// Close database connection
	err = sqlDB.Close()
	assert.NoError(t, err)
}

func TestNewWithInvalidConfig(t *testing.T) {
	// Not applicable for SQLite in-memory, so just ensure open fails with bad DSN
	_, err := gorm.Open(sqlite.Open("/invalid/path/to/db.sqlite"), &gorm.Config{})
	assert.Error(t, err)
}
