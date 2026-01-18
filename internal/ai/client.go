package ai

import (
	"context"
)

// Client defines the interface for AI answer generation
type Client interface {
	GenerateAnswer(ctx context.Context, query string) (string, error)
}
