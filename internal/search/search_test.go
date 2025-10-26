package search

import (
	"fmt"
	"testing"

	"hpi-search-engine/internal/data"
	"hpi-search-engine/internal/index"
	"hpi-search-engine/internal/text"
)

func buildIndexForTests(t *testing.T, docs []data.Document) index.InvertedIndex {
	t.Helper()
	tokenizer := text.NewTokenizer()
	return index.Build(docs, tokenizer)
}

func TestSearchReturnsErrorOnEmptyQuery(t *testing.T) {
	idx := make(index.InvertedIndex)
	tokenizer := text.NewTokenizer()

	_, err := Search(idx, tokenizer, "")
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

	results, err := Search(idx, tokenizer, "search engine")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	results, err := Search(idx, tokenizer, "missing token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results when token is missing, got %v", results)
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

	results, err := Search(idx, tokenizer, "term")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}
}
