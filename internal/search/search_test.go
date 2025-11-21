package search

import (
	"slices"
	"strings"
	"testing"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/processing"
	"luen-search-engine/internal/text"
)

func TestSearchReturnsErrorOnEmptyQuery(t *testing.T) {
	idx := make(index.InvertedIndex)
	tokenizer := text.NewTokenizer()

	_, _, err := Search(processing.NewMemorySource(idx), tokenizer, "")
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
	if err.Error() != "query must not be empty" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSearchSingleTermRanksByFrequency(t *testing.T) {
	docs := []data.Document{
		{ID: 123, Title: "Alpha", Text: "Search engine engine"},
		{ID: 456, Title: "Beta", Text: "Engine search"},
		{ID: 789, Title: "Gamma", Text: "Search search engine"},
	}

	tokenizer := text.NewTokenizer()
	idx := index.Build(docs, tokenizer)

	results, total, err := Search(processing.NewMemorySource(idx), tokenizer, "search engine")
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
		{ID: 123, Title: "Alpha", Text: "search term"},
	}

	tokenizer := text.NewTokenizer()
	idx := index.Build(docs, tokenizer)

	results, total, err := Search(processing.NewMemorySource(idx), tokenizer, "missing token")
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

	tokenizer := text.NewTokenizer()
	idx := index.Build(docs, tokenizer)

	results, total, err := Search(processing.NewMemorySource(idx), tokenizer, "term")
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

func TestSearchWithNotOperator(t *testing.T) {
	docs := []data.Document{
		{ID: 123, Title: "Alpha", Text: "cat dog"},
		{ID: 456, Title: "Beta", Text: "cat bird"},
		{ID: 789, Title: "Gamma", Text: "cat dog bird"},
	}

	tokenizer := text.NewTokenizer()
	idx := index.Build(docs, tokenizer)

	results, total, err := Search(processing.NewMemorySource(idx), tokenizer, "cat and not dog")
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

func TestSearchOrPrecedence(t *testing.T) {
	docs := []data.Document{
		{ID: 1, Title: "Doc1", Text: "cat dog"},
		{ID: 2, Title: "Doc2", Text: "bird"},
		{ID: 3, Title: "Doc3", Text: "cat bird"},
		{ID: 4, Title: "Doc4", Text: "dog"},
	}

	tokenizer := text.NewTokenizer()
	idx := index.Build(docs, tokenizer)

	results, total, err := Search(processing.NewMemorySource(idx), tokenizer, "cat and dog or bird")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 total matches, got %d", total)
	}

	want := []uint32{1, 2, 3}
	if len(results) != len(want) {
		t.Fatalf("expected %d results, got %d", len(want), len(results))
	}
	for i, docID := range want {
		if results[i].DocID != docID {
			t.Fatalf("result %d: got docID %d, want %d", i, results[i].DocID, docID)
		}
	}
}

func TestSearchParenthesesOverridePrecedence(t *testing.T) {
	docs := []data.Document{
		{ID: 1, Title: "Doc1", Text: "cat bird"},
		{ID: 2, Title: "Doc2", Text: "dog bird"},
		{ID: 3, Title: "Doc3", Text: "bird"},
	}

	tokenizer := text.NewTokenizer()
	idx := index.Build(docs, tokenizer)

	results, total, err := Search(processing.NewMemorySource(idx), tokenizer, "(cat or dog) and bird")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 matches, got %d", total)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].DocID != 1 || results[1].DocID != 2 {
		t.Fatalf("unexpected doc order: %+v", results)
	}
}

func TestSearchPhraseQueryMatchesContiguousTokens(t *testing.T) {
	docs := []data.Document{
		{ID: 123, Title: "", Text: "quick brown fox"},
		{ID: 456, Title: "", Text: "quick fox brown"},
		{ID: 789, Title: "", Text: "quick brown quick brown"},
	}

	tokenizer := text.NewTokenizer()
	idx := index.Build(docs, tokenizer)

	results, total, err := Search(processing.NewMemorySource(idx), tokenizer, "\"quick brown\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 2 {
		t.Fatalf("expected total results to be 2 (ID 123 and 789), got %d", total)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].DocID != 789 || results[0].TotalTermFrequency != 2 {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
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

	if results[1].DocID != 123 || results[1].TotalTermFrequency != 1 {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
}

func TestSearchOnlyNegationsReturnError(t *testing.T) {
	docs := []data.Document{
		{ID: 1, Title: "Doc1", Text: "cat"},
	}

	tokenizer := text.NewTokenizer()
	idx := index.Build(docs, tokenizer)

	_, _, err := Search(processing.NewMemorySource(idx), tokenizer, "not cat")
	if err == nil {
		t.Fatal("expected error for negation-only query, got nil")
	}
	if !strings.Contains(err.Error(), "NOT expressions must be combined with a positive search term") {
		t.Fatalf("unexpected error: %v", err)
	}
}
