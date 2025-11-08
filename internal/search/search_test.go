package search

import (
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
		{ID: 123, Title: "Alpha", Text: "Search engine engine"},
		{ID: 456, Title: "Beta", Text: "Engine search"},
		{ID: 789, Title: "Gamma", Text: "Search search engine"},
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

	if results[0].DocID != 123 || results[0].TotalTermFrequency != 3 {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	if results[1].DocID != 789 || results[1].TotalTermFrequency != 3 {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
	if results[2].DocID != 456 || results[2].TotalTermFrequency != 2 {
		t.Fatalf("unexpected third result: %+v", results[2])
	}
}

func TestSearchMissingTokenReturnsNil(t *testing.T) {
	docs := []data.Document{
		{ID: 123, Title: "Alpha", Text: "Search term"},
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
			ID:    uint32(i),
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
		{ID: 123, Title: "Alpha", Text: "cat dog"},
		{ID: 456, Title: "Beta", Text: "cat bird"},
		{ID: 789, Title: "Gamma", Text: "cat dog bird"},
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

	if results[0].DocID != 456 {
		t.Fatalf("expected 456 (has cat but not dog), got %v", results[0].DocID)
	}
}

func TestSearchWithOnlyNegatedTokensReturnsError(t *testing.T) {
	docs := []data.Document{
		{ID: 123, Title: "Alpha", Text: "cat dog"},
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
		{ID: 123, Title: "Alpha", Text: "cat dog"},
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
		{ID: 123, Title: "Alpha", Text: "cat dog bird"},
		{ID: 456, Title: "Beta", Text: "cat fish"},
		{ID: 789, Title: "Gamma", Text: "cat dog"},
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

	if results[0].DocID != 456 {
		t.Fatalf("expected 456 (has cat but not dog or bird), got %v", results[0].DocID)
	}
}

func TestSearchPhraseQueryMatchesContiguousTokens(t *testing.T) {
	docs := []data.Document{
		{ID: 123, Title: "", Text: "quick brown fox"},
		{ID: 456, Title: "", Text: "quick fox brown"},
		{ID: 789, Title: "", Text: "quick brown quick brown"},
	}

	idx := buildIndexForTests(t, docs)
	tokenizer := text.NewTokenizer()

	results, total, err := Search(idx, tokenizer, "quick brown", "phrase")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 2 {
		t.Fatalf("expected total results to be 2 (ID 123 and 789), got %d", total)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// 789 should come first because it contains the phrase twice
	if results[0].DocID != 789 || results[0].TotalTermFrequency != 2 {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	// check positions for 789: quick at positions 0 and 2, brown at 1 and 3
	if len(results[0].Matches) < 2 {
		t.Fatalf("expected at least 2 token matches in 789, got %v", results[0].Matches)
	}
	head := results[0].Matches[0]
	if head.Token != "quick" || head.Frequency != 2 || !slices.Equal(head.Positions, []int{0, 2}) {
		t.Fatalf("unexpected head match for 789: %+v", head)
	}
	sec := results[0].Matches[1]
	if sec.Token != "brown" || sec.Frequency != 2 || !slices.Equal(sec.Positions, []int{1, 3}) {
		t.Fatalf("unexpected second match for 789: %+v", sec)
	}

	// 123 should be the second result with a single occurrence
	if results[1].DocID != 123 || results[1].TotalTermFrequency != 1 {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
}
