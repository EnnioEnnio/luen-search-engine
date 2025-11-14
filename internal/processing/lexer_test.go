package processing

import (
	"testing"

	"luen-search-engine/internal/text"
)

func TestLexerTokenizesOperatorsAndPhrases(t *testing.T) {
	tokenizer := text.NewTokenizer()
	lexer := newLexer(`alpha and (beta or not "gamma delta")`, tokenizer)

	tokens, err := lexer.tokens()
	if err != nil {
		t.Fatalf("unexpected lexer error: %v", err)
	}

	gotTypes := make([]tokenType, len(tokens))
	gotValues := make([]string, len(tokens))
	for i, tok := range tokens {
		gotTypes[i] = tok.typ
		gotValues[i] = tok.value
	}

	wantTypes := []tokenType{
		tTerm, tAnd, tLParen, tTerm, tOr, tNot, tLPhrase, tTerm, tTerm, tRPhrase, tRParen,
	}
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("expected %d tokens, got %d (%v)", len(wantTypes), len(gotTypes), gotTypes)
	}
	for i := range wantTypes {
		if gotTypes[i] != wantTypes[i] {
			t.Fatalf("token %d: expected type %v, got %v", i, wantTypes[i], gotTypes[i])
		}
	}

	wantValues := []string{"alpha", "and", "(", "beta", "or", "not", "\"", "gamma", "delta", "\"", ")"}
	for i := range wantValues {
		if gotValues[i] != wantValues[i] {
			t.Fatalf("token %d: expected value %q, got %q", i, wantValues[i], gotValues[i])
		}
	}
}

func TestLexerCapturesTrailingTerm(t *testing.T) {
	tokenizer := text.NewTokenizer()
	lexer := newLexer("single", tokenizer)

	tokens, err := lexer.tokens()
	if err != nil {
		t.Fatalf("unexpected lexer error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token, got %d", len(tokens))
	}
	if tokens[0].typ != tTerm || tokens[0].value != "single" {
		t.Fatalf("unexpected token: %+v", tokens[0])
	}
}
