package search

import (
	"fmt"
	"slices"
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

	_, _, err := Search(idx, tokenizer, "", "single")
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

	results, total, err := Search(idx, tokenizer, "search engine", "single")
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

	results, total, err := Search(idx, tokenizer, "missing token", "")
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

	results, total, err := Search(idx, tokenizer, "term", "")
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

	results, total, err := Search(idx, tokenizer, "cat -dog", "")
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

	_, _, err := Search(idx, tokenizer, "-cat -dog", "")
	if err == nil {
		t.Fatal("expected error for query with only negations, got nil")
	}
	if !strings.Contains(err.Error(), "query contains only negations") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSearchWithORAndNegationReturnsError(t *testing.T) {
	docs := []data.Document{
		{ID: "docA", Title: "Alpha", Text: "cat dog"},
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	_, _, err := Search(idx, tokenizer, "cat or -dog", "")
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

	results, total, err := Search(idx, tokenizer, "cat -dog -bird", "")
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

func TestSearchPhraseQueryMatchesContiguousTokens(t *testing.T) {
	docs := []data.Document{
		{ID: "docA", Title: "", Text: "quick brown fox"},
		{ID: "docB", Title: "", Text: "quick fox brown"},
		{ID: "docC", Title: "", Text: "quick brown quick brown"},
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	results, total, err := Search(idx, tokenizer, "quick brown", "phrase")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 2 {
		t.Fatalf("expected total results to be 2 (docA and docC), got %d", total)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// docC should come first because it contains the phrase twice
	if results[0].DocID != "docC" || results[0].TotalTermFrequency != 2 {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	// check positions for docC: quick at positions 0 and 2, brown at 1 and 3
	if len(results[0].Matches) < 2 {
		t.Fatalf("expected at least 2 token matches in docC, got %v", results[0].Matches)
	}
	head := results[0].Matches[0]
	if head.Token != "quick" || head.Frequency != 2 || !slices.Equal(head.Positions, []int{0, 2}) {
		t.Fatalf("unexpected head match for docC: %+v", head)
	}
	sec := results[0].Matches[1]
	if sec.Token != "brown" || sec.Frequency != 2 || !slices.Equal(sec.Positions, []int{1, 3}) {
		t.Fatalf("unexpected second match for docC: %+v", sec)
	}

	// docA should be the second result with a single occurrence
	if results[1].DocID != "docA" || results[1].TotalTermFrequency != 1 {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
}
