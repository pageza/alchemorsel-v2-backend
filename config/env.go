package config

import (
	"os"
)

// Environment represents the current runtime environment
type Environment string

const (
	Development Environment = "development"
	Test        Environment = "test"
	CI          Environment = "ci"
	Production  Environment = "production"
)

// GetEnvironment determines the current environment
func GetEnvironment() Environment {
	// CI environment is automatically detected
	if os.Getenv("CI") == "true" {
		return CI
	}

	// Other environments are set via ENV variable
	switch env := os.Getenv("ENV"); env {
	case "production":
		return Production
	case "test":
		return Test
	case "development":
		return Development
	default:
		return Development // Default to development
	}
}

// IsDevelopment returns true if the current environment is development
func IsDevelopment() bool {
	return GetEnvironment() == Development
}

// IsTest returns true if the current environment is test
func IsTest() bool {
	return GetEnvironment() == Test
}

// IsCI returns true if the current environment is CI
func IsCI() bool {
	return GetEnvironment() == CI
}

// IsProduction returns true if the current environment is production
func IsProduction() bool {
	return GetEnvironment() == Production
}
