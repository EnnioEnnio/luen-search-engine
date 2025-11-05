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
func NewTokenizer() Tokenizer {
	stop := []string{
		"a", "an", "and", "are", "as", "at", "be", "but", "by", "for", "if", "in",
		"into", "is", "it", "no", "not", "of", "on", "or", "such", "that", "the",
		"their", "then", "there", "these", "they", "this", "to", "was", "will", "with",
	}

	stopSet := make(map[string]struct{}, len(stop))
	for _, token := range stop {
		stopSet[token] = struct{}{}
	}

	return Tokenizer{stopwords: stopSet}
}

// Tokenize returns normalized tokens extracted from the provided string.
func (t Tokenizer) Tokenize(value string) []string {
	if value == "" {
		return nil
	}

	// Slice for keeping track of negated values (bool true)
	isTokenNegated := make([]bool, 0)

	lower := strings.ToLower(value)
	tokens := make([]string, 0)
	var builder strings.Builder

	flush := func() {
		if builder.Len() == 0 {
			return
		}
		token := builder.String()
		// isNegated checken strings.prefix ... -> slice befüllen 
		builder.Reset()
		if _, isStop := t.stopwords[token]; isStop {
			return
		}
		tokens = append(tokens, token)
	}

	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || // or is minus {
			builder.WriteRune(r)
			continue
		}
		flush()
	}

	flush()
	return tokens
}
