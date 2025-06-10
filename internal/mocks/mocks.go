package mocks

import (
	"github.com/pageza/alchemorsel-v2/backend/internal/types"
	"github.com/pgvector/pgvector-go"
)

// MockEmbeddingService is a mock implementation of the embedding service
type MockEmbeddingService struct{}

func (m *MockEmbeddingService) GenerateEmbedding(text string) (pgvector.Vector, error) {
	return pgvector.NewVector([]float32{0.1, 0.2, 0.3}), nil
}

func (m *MockEmbeddingService) GenerateEmbeddingFromRecipe(recipe *types.Recipe) (pgvector.Vector, error) {
	return pgvector.NewVector([]float32{0.1, 0.2, 0.3}), nil
}

// ... existing code ...
