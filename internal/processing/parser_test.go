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

// TestPruneNotNodePreservesExclusion tests that NOT nodes are not pruned away
// even when their child terms have low IDF. This is a regression test for a bug
// where NotNode.Prune() was incorrectly pruning its child, which could cause the
// NOT node itself to be removed if the child returned nil.
func TestPruneNotNodePreservesExclusion(t *testing.T) {
	// Parse a query with a NOT operator: "alpha and not beta"
	node, err := Parse("alpha and not beta", newTestTokenizer())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Create a highIDFTokens map that only includes "alpha" (beta has low IDF)
	highIDFTokens := map[Token]bool{
		"alpha": true,
		// "beta" is intentionally omitted - it has low IDF
	}

	// Prune the AST
	prunedNode := node.Prune(highIDFTokens)

	// The pruned node should still be an AndNode
	andNode, ok := prunedNode.(*AndNode)
	if !ok {
		t.Fatalf("expected AndNode after pruning, got %T", prunedNode)
	}

	// The AndNode should have 2 children: alpha term and NOT beta
	if len(andNode.Children) != 2 {
		t.Fatalf("expected 2 children after pruning, got %d", len(andNode.Children))
	}

	// First child should be the alpha term
	checkTerm(t, andNode.Children[0], "alpha")

	// Second child should be the NOT node (not pruned away)
	notNode, ok := andNode.Children[1].(*NotNode)
	if !ok {
		t.Fatalf("expected NotNode as second child after pruning, got %T", andNode.Children[1])
	}

	// The NOT node should still have its child (beta term), even though beta has low IDF
	if notNode.Child == nil {
		t.Fatal("NOT node child was pruned away - this is the regression!")
	}

	checkTerm(t, notNode.Child, "beta")
}

// TestPruneTermNodeWithLowIDF tests that term nodes with low IDF are pruned
func TestPruneTermNodeWithLowIDF(t *testing.T) {
	// Parse a simple term
	node, err := Parse("beta", newTestTokenizer())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Create a highIDFTokens map that does not include "beta"
	highIDFTokens := map[Token]bool{
		"alpha": true,
	}

	// Prune the AST
	prunedNode := node.Prune(highIDFTokens)

	// The low-IDF term should be pruned to nil
	if prunedNode != nil {
		t.Fatalf("expected nil after pruning low-IDF term, got %T", prunedNode)
	}
}

// TestPruneAndNodeRemovesLowIDFTerms tests that AND nodes remove low-IDF children
func TestPruneAndNodeRemovesLowIDFTerms(t *testing.T) {
	// Parse: "alpha and beta and gamma"
	node, err := Parse("alpha and beta and gamma", newTestTokenizer())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Only alpha and gamma have high IDF
	highIDFTokens := map[Token]bool{
		"alpha": true,
		"gamma": true,
	}

	// Prune the AST
	prunedNode := node.Prune(highIDFTokens)

	// Should still be an AndNode
	andNode, ok := prunedNode.(*AndNode)
	if !ok {
		t.Fatalf("expected AndNode after pruning, got %T", prunedNode)
	}

	// Should have 2 children (beta was pruned)
	if len(andNode.Children) != 2 {
		t.Fatalf("expected 2 children after pruning beta, got %d", len(andNode.Children))
	}

	checkTerm(t, andNode.Children[0], "alpha")
	checkTerm(t, andNode.Children[1], "gamma")
}

// TestPruneOrNodeRemovesLowIDFTerms tests that OR nodes remove low-IDF children
func TestPruneOrNodeRemovesLowIDFTerms(t *testing.T) {
	// Parse: "alpha or beta or gamma"
	node, err := Parse("alpha or beta or gamma", newTestTokenizer())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Only alpha has high IDF
	highIDFTokens := map[Token]bool{
		"alpha": true,
	}

	// Prune the AST
	prunedNode := node.Prune(highIDFTokens)

	// After pruning, only alpha remains, so the OrNode should collapse to a single TermNode
	term, ok := prunedNode.(*TermNode)
	if !ok {
		t.Fatalf("expected TermNode after pruning OR node to single term, got %T", prunedNode)
	}

	if term.Token != "alpha" {
		t.Fatalf("expected token alpha, got %q", term.Token)
	}
}

// TestPruneComplexQueryWithNot tests a complex query with NOT to ensure exclusions are preserved
func TestPruneComplexQueryWithNot(t *testing.T) {
	// Parse: "(alpha or beta) and not gamma"
	node, err := Parse("(alpha or beta) and not gamma", newTestTokenizer())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Only alpha has high IDF (beta and gamma are low IDF)
	highIDFTokens := map[Token]bool{
		"alpha": true,
	}

	// Prune the AST
	prunedNode := node.Prune(highIDFTokens)

	// Should be an AndNode
	andNode, ok := prunedNode.(*AndNode)
	if !ok {
		t.Fatalf("expected AndNode after pruning, got %T", prunedNode)
	}

	// Should have 2 children: the alpha term (OR collapsed) and the NOT node
	if len(andNode.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(andNode.Children))
	}

	// First child should be alpha (the OR node collapsed to a single term)
	checkTerm(t, andNode.Children[0], "alpha")

	// Second child should be the NOT node (preserved even though gamma has low IDF)
	notNode, ok := andNode.Children[1].(*NotNode)
	if !ok {
		t.Fatalf("expected NotNode as second child, got %T", andNode.Children[1])
	}

	// The NOT node should still have its child
	if notNode.Child == nil {
		t.Fatal("NOT node child was incorrectly pruned")
	}

	checkTerm(t, notNode.Child, "gamma")
}
