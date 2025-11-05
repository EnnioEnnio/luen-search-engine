package search

import (
	"fmt"
	"strings"
	"testing"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/text"
)

func buildIndexForTests(t *testing.T, docs []data.Document) index.InvertedIndex {
	t.Helper()
	tokenizer := text.NewTokenizer()
	return index.Build(docs, tokenizer)
}

func TestSearchReturnsErrorOnEmptyQuery(t *testing.T) {
	idx := make(index.InvertedIndex)
	tokenizer := text.NewTokenizer()

	_, _, err := Search(idx, tokenizer, "")
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
	if err.Error() != "query must not be empty" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSearchReturnsResultsSortedByScore(t *testing.T) {
	docs := []data.Document{
		{ID: "docA", Title: "Alpha", Text: "Search engine engine"},
		{ID: "docB", Title: "Beta", Text: "Engine search"},
		{ID: "docC", Title: "Gamma", Text: "Search search engine"},
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	results, total, err := Search(idx, tokenizer, "search engine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 3 {
		t.Fatalf("expected total results to be 3, got %d", total)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].DocID != "docA" || results[0].TotalTermFrequency != 3 {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	if results[1].DocID != "docC" || results[1].TotalTermFrequency != 3 {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
	if results[2].DocID != "docB" || results[2].TotalTermFrequency != 2 {
		t.Fatalf("unexpected third result: %+v", results[2])
	}
}

func TestSearchMissingTokenReturnsNil(t *testing.T) {
	docs := []data.Document{
		{ID: "docA", Title: "Alpha", Text: "Search term"},
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	results, total, err := Search(idx, tokenizer, "missing token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results when token is missing, got %v", results)
	}
	if total != 0 {
		t.Fatalf("expected total results to be 0, got %d", total)
	}
}

func TestSearchLimitsToTopTenResults(t *testing.T) {
	docs := make([]data.Document, 11)
	for i := range docs {
		docs[i] = data.Document{
			ID:    fmt.Sprintf("doc%02d", i),
			Title: "Title",
			Text:  "Term term",
		}
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	results, total, err := Search(idx, tokenizer, "term")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}
	if total <= 10 {
		t.Fatalf("expected total results to exceed 10, got %d", total)
	}
}

func TestSearchWithNegatedToken(t *testing.T) {
	docs := []data.Document{
		{ID: "docA", Title: "Alpha", Text: "cat dog"},
		{ID: "docB", Title: "Beta", Text: "cat bird"},
		{ID: "docC", Title: "Gamma", Text: "cat dog bird"},
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	results, total, err := Search(idx, tokenizer, "cat -dog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 1 {
		t.Fatalf("expected total results to be 1, got %d", total)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].DocID != "docB" {
		t.Fatalf("expected docB (has cat but not dog), got %s", results[0].DocID)
	}
}

func TestSearchWithOnlyNegatedTokensReturnsError(t *testing.T) {
	docs := []data.Document{
		{ID: "docA", Title: "Alpha", Text: "cat dog"},
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	_, _, err := Search(idx, tokenizer, "-cat -dog")
	if err == nil {
		t.Fatal("expected error for query with only negations, got nil")
	}
	if !strings.Contains(err.Error(), "Only negations detected") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSearchWithORAndNegationReturnsError(t *testing.T) {
	docs := []data.Document{
		{ID: "docA", Title: "Alpha", Text: "cat dog"},
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	_, _, err := Search(idx, tokenizer, "cat or -dog")
	if err == nil {
		t.Fatal("expected error for OR query with negation, got nil")
	}
	if !strings.Contains(err.Error(), "combining NOT and OR queries is not allowed") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSearchWithMultipleNegatedTokens(t *testing.T) {
	docs := []data.Document{
		{ID: "docA", Title: "Alpha", Text: "cat dog bird"},
		{ID: "docB", Title: "Beta", Text: "cat fish"},
		{ID: "docC", Title: "Gamma", Text: "cat dog"},
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	results, total, err := Search(idx, tokenizer, "cat -dog -bird")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 1 {
		t.Fatalf("expected total results to be 1, got %d", total)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].DocID != "docB" {
		t.Fatalf("expected docB (has cat but not dog or bird), got %s", results[0].DocID)
	}
}
