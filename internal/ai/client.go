package ai

import (
	"context"
)

// Client defines the interface for AI answer generation
type Client interface {
	GenerateAnswer(ctx context.Context, query string, documents []Document) (string, error)
}

// Document represents a search result that will be used as context for the AI
type Document struct {
	Title   string
	Content string
	Score   float64
}
