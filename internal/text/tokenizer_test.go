package text

import (
	"reflect"
	"testing"
)

func TestTokenizeNormalizesAndFilters(t *testing.T) {
	tokenizer := NewTokenizer()
	input := "The QUICK brown-fox 123 -jumps, and the dog."
	want := []string{"quick", "brown-fox", "123", "jumps", "dog"}

	if got, _ := tokenizer.Tokenize(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("Tokenize(%q) = %v, want %v", input, got, want)
	}
}

func TestTokenizeEmptyString(t *testing.T) {
	tokenizer := NewTokenizer()
	if tokens, _ := tokenizer.Tokenize(""); tokens != nil {
		t.Fatalf("expected nil tokens for empty input, got %v", tokens)
	}
}
