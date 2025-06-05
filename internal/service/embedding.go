package service

import (
	"strings"

	pgvector "github.com/pgvector/pgvector-go"
)

// GenerateEmbedding returns a simple deterministic embedding for the given text.
// This implementation counts the total length, vowels and consonants.
func GenerateEmbedding(text string) pgvector.Vector {
	text = strings.ToLower(text)
	var vowels, consonants float32
	for _, r := range text {
		if strings.ContainsRune("aeiou", r) {
			vowels++
		} else if r >= 'a' && r <= 'z' {
			consonants++
		}
	}
	length := float32(len(text))
	return pgvector.NewVector([]float32{length, vowels, consonants})
}
