package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIClient implements the Client interface using OpenAI's API
type OpenAIClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
		model:  "gpt-4o-mini", // Schneller und günstiger als gpt-4
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// OpenAI API Request/Response Structures
type openAIRequest struct {
	Model     string    `json:"model"`
	Messages  []message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// GenerateAnswer generates an AI answer based on the query and top documents
func (c *OpenAIClient) GenerateAnswer(ctx context.Context, query string, documents []Document) (string, error) {
	if len(documents) == 0 {
		return "", fmt.Errorf("no documents provided")
	}

	// Baue den Kontext aus den Top-Dokumenten
	contextBuilder := strings.Builder{}
	contextBuilder.WriteString("Here are the most relevant documents:\n\n")

	// Nehme nur die Top 3 Dokumente für den Kontext (um Token-Limit nicht zu überschreiten)
	maxDocs := 3
	if len(documents) < maxDocs {
		maxDocs = len(documents)
	}

	for i := 0; i < maxDocs; i++ {
		doc := documents[i]
		contextBuilder.WriteString(fmt.Sprintf("Document %d:\n", i+1))
		contextBuilder.WriteString(fmt.Sprintf("Title: %s\n", doc.Title))

		// Begrenze den Content auf ~500 Zeichen pro Dokument
		content := doc.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		contextBuilder.WriteString(fmt.Sprintf("Content: %s\n\n", content))
	}

	// Erstelle den System-Prompt
	systemPrompt := `You are a helpful search assistant. Your task is to provide a concise, accurate answer based on the provided documents. 
Keep your answer to 2-3 sentences maximum. Be direct and informative. If the documents don't contain enough information to answer the question, say so briefly.`

	// Erstelle den User-Prompt
	userPrompt := fmt.Sprintf("%s\n\nQuestion: %s", contextBuilder.String(), query)

	// Baue den API Request
	reqBody := openAIRequest{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens: 150, // Begrenze die Antwort auf ~100-150 Tokens (2-3 Sätze)
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Sende Request an OpenAI API
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse Response
	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return openAIResp.Choices[0].Message.Content, nil
}
