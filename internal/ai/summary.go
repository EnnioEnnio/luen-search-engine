package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const DefaultSummaryModel = "gpt-4o-mini"

type SummaryResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func GenerateSummary(ctx context.Context, apiKey, model, query string, results []SummaryResult) (string, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", errors.New("OPENAI_API_KEY is not set")
	}
	if model == "" {
		model = DefaultSummaryModel
	}

	sourceSummary := buildSourceSummary(results)
	systemPrompt := "You are a concise search assistant. Using only the provided sources, answer the query in 2-3 sentences. If sources are insufficient, say so."
	userPrompt := fmt.Sprintf("Query: %s\n\nSources:\n%s", query, sourceSummary)

	payload := chatCompletionRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		MaxTokens:   200,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal summary request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create summary request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send summary request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("summary request failed with status %s", resp.Status)
	}

	var parsed chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode summary response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", errors.New("summary response missing choices")
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func buildSourceSummary(results []SummaryResult) string {
	if len(results) == 0 {
		return "No sources available."
	}

	var builder strings.Builder
	maxSources := len(results)
	if maxSources > 6 {
		maxSources = 6
	}
	for i := 0; i < maxSources; i++ {
		result := results[i]
		content := strings.TrimSpace(result.Content)
		if len(content) > 280 {
			content = content[:280] + "..."
		}
		fmt.Fprintf(&builder, "%d. %s\nURL: %s\nSnippet: %s\n\n", i+1, result.Title, result.URL, content)
	}
	return strings.TrimSpace(builder.String())
}
