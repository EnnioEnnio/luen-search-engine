package ai

import (
	"context"
)

type Client interface {
	GenerateAnswer(ctx context.Context, query string) (string, error)
}
