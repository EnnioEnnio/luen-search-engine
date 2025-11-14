package processing

import (
	"strings"
	"testing"

	"luen-search-engine/internal/text"
)

func newTestTokenizer() *text.Tokenizer {
	return text.NewTokenizer()
}

func TestParseSingleTermProducesTermNode(t *testing.T) {
	node, err := Parse("alpha", newTestTokenizer())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	term, ok := node.(*TermNode)
	if !ok {
		t.Fatalf("expected TermNode, got %T", node)
	}
	if term.Token != "alpha" {
		t.Fatalf("expected token alpha, got %q", term.Token)
	}
}

func TestParseOperatorPrecedence(t *testing.T) {
	node, err := Parse("alpha and beta or gamma", newTestTokenizer())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	orNode, ok := node.(*OrNode)
	if !ok {
		t.Fatalf("expected OrNode at root, got %T", node)
	}
	if len(orNode.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(orNode.Children))
	}

	andNode, ok := orNode.Children[0].(*AndNode)
	if !ok {
		t.Fatalf("expected first child to be AndNode, got %T", orNode.Children[0])
	}
	if len(andNode.Children) != 2 {
		t.Fatalf("expected AndNode to have 2 children, got %d", len(andNode.Children))
	}

	checkTerm(t, andNode.Children[0], "alpha")
	checkTerm(t, andNode.Children[1], "beta")
	checkTerm(t, orNode.Children[1], "gamma")
}

func TestParseParentheses(t *testing.T) {
	node, err := Parse("(alpha or beta) and gamma", newTestTokenizer())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	andNode, ok := node.(*AndNode)
	if !ok {
		t.Fatalf("expected AndNode at root, got %T", node)
	}
	if len(andNode.Children) != 2 {
		t.Fatalf("expected AndNode to have 2 children, got %d", len(andNode.Children))
	}

	orNode, ok := andNode.Children[0].(*OrNode)
	if !ok {
		t.Fatalf("expected first child to be OrNode, got %T", andNode.Children[0])
	}
	if len(orNode.Children) != 2 {
		t.Fatalf("expected OrNode to have 2 children, got %d", len(orNode.Children))
	}
	checkTerm(t, orNode.Children[0], "alpha")
	checkTerm(t, orNode.Children[1], "beta")
	checkTerm(t, andNode.Children[1], "gamma")
}

func TestParsePhraseAndNot(t *testing.T) {
	node, err := Parse("\"new york\" and not pizza", newTestTokenizer())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	andNode, ok := node.(*AndNode)
	if !ok {
		t.Fatalf("expected AndNode at root, got %T", node)
	}
	if len(andNode.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(andNode.Children))
	}

	phrase, ok := andNode.Children[0].(*PhraseNode)
	if !ok {
		t.Fatalf("expected first child to be PhraseNode, got %T", andNode.Children[0])
	}
	if !slicesEqual(phrase.Tokens, []string{"new", "york"}) {
		t.Fatalf("unexpected phrase tokens: %v", phrase.Tokens)
	}

	notNode, ok := andNode.Children[1].(*NotNode)
	if !ok {
		t.Fatalf("expected second child to be NotNode, got %T", andNode.Children[1])
	}
	checkTerm(t, notNode.Child, "pizza")
}

func TestParseReportsUnmatchedParenthesis(t *testing.T) {
	_, err := Parse("(alpha or beta", newTestTokenizer())
	if err == nil {
		t.Fatal("expected parse error for unmatched parenthesis, got nil")
	}
	if !strings.Contains(err.Error(), "expected closing ')'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseReportsUnterminatedPhrase(t *testing.T) {
	_, err := Parse("\"new york", newTestTokenizer())
	// lexer will still emit tokens, parser should fail expecting closing quote
	if err == nil {
		t.Fatal("expected parse error for unterminated phrase, got nil")
	}
	if !strings.Contains(err.Error(), "unterminated phrase") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseReportsEmptyQuery(t *testing.T) {
	_, err := Parse("   ", newTestTokenizer())
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
	if !strings.Contains(err.Error(), "no searchable terms") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func checkTerm(t *testing.T, node Node, want string) {
	t.Helper()
	term, ok := node.(*TermNode)
	if !ok {
		t.Fatalf("expected TermNode, got %T", node)
	}
	if term.Token != want {
		t.Fatalf("expected term %q, got %q", want, term.Token)
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
