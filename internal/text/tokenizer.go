package text

import (
	"strings"
	"unicode"
)

// Tokenizer converts input text into normalized tokens suited for indexing.
type Tokenizer struct {
	stopwords map[string]struct{}
}

// NewTokenizer creates a tokenizer with a basic English stopword list.
func NewTokenizer() *Tokenizer {
	stop := []string{
		"a", "an", "and", "are", "as", "at", "be", "but", "by", "for", "if", "in",
		"into", "is", "it", "no", "not", "of", "on", "or", "such", "that", "the",
		"their", "then", "there", "these", "they", "this", "to", "was", "will", "with",
	}

	stopSet := make(map[string]struct{}, len(stop))
	for _, token := range stop {
		stopSet[token] = struct{}{}
	}

	return &Tokenizer{stopwords: stopSet}
}

// Tokenize returns normalized tokens extracted from the provided string.
func (t *Tokenizer) Tokenize(value string) []string {
	if t == nil || value == "" {
		return nil
	}

	lower := strings.ToLower(value)
	tokens := make([]string, 0)
	var builder strings.Builder

	flush := func() {
		if builder.Len() == 0 {
			return
		}
		token := builder.String()
		builder.Reset()
		isNegated := strings.HasPrefix(token, "-")

		if isNegated {
			token = token[1:]
		}
		if _, isStop := t.stopwords[token]; isStop {
			return
		}
		tokens = append(tokens, token)
	}

	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			continue
		}
		flush()
	}

	flush()
	return tokens
}

// Convenience wrapper around Tokenize() for single terms (used in query tokenization)
func (t *Tokenizer) TokenizeTerm(value string) string {
	tokenized := t.Tokenize(value)

	if len(tokenized) == 0 {
		return ""
	}

	return tokenized[0]
}
