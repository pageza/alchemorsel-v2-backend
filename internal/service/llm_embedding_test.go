package service

import (
	"testing"

	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmbeddingService for testing
type MockEmbeddingService struct {
	mock.Mock
}

func (m *MockEmbeddingService) GenerateEmbedding(text string) (pgvector.Vector, error) {
	args := m.Called(text)
	return args.Get(0).(pgvector.Vector), args.Error(1)
}

func (m *MockEmbeddingService) GenerateEmbeddingFromRecipe(name, description string, ingredients []string, category string, dietary []string) (pgvector.Vector, error) {
	args := m.Called(name, description, ingredients, category, dietary)
	return args.Get(0).(pgvector.Vector), args.Error(1)
}

func TestEmbeddingGenerationInMultiCall(t *testing.T) {
	t.Run("Embedding logic verification", func(t *testing.T) {
		// Test the embedding empty check logic
		var emptyDraft RecipeDraft
		
		// Check empty embedding detection
		var embeddingEmpty bool
		if embeddingSlice := emptyDraft.Embedding.Slice(); len(embeddingSlice) == 0 {
			embeddingEmpty = true
		}
		assert.True(t, embeddingEmpty, "Empty draft should be detected as having no embedding")

		// Test draft with embedding
		vectorWithData := pgvector.NewVector([]float32{0.1, 0.2, 0.3})
		draftWithEmbedding := RecipeDraft{
			Embedding: vectorWithData,
		}
		
		embeddingEmpty = false
		if embeddingSlice := draftWithEmbedding.Embedding.Slice(); len(embeddingSlice) == 0 {
			embeddingEmpty = true
		}
		assert.False(t, embeddingEmpty, "Draft with embedding should not be detected as empty")
		assert.Equal(t, []float32{0.1, 0.2, 0.3}, draftWithEmbedding.Embedding.Slice())
	})

}