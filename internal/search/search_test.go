package search

import (
	"context"
	"log"
	"os"
	"slices"
	"strings"
	"testing"

	"luen-search-engine/internal/index"
	"luen-search-engine/internal/index/disk"
	"luen-search-engine/internal/indexer"
	"luen-search-engine/internal/model"
	"luen-search-engine/internal/text"
)

var (
	testPostingStore *disk.PostingStore
	testTokenizer    *text.Tokenizer
	testDocLengths   map[model.DocID]model.FieldDocLengths
)

func TestMain(m *testing.M) {
	// Setup
	testTokenizer = text.NewTokenizer()
	tempDir, err := os.MkdirTemp("", "search_test_index")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Build index from testing.tsv
	// We assume testing.tsv is in ../../data/testing.tsv relative to this test file
	dataPath := "../../data/testing.tsv"
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		// Fallback for running from root
		dataPath = "data/testing.tsv"
	}

	builder := indexer.NewBuilder(indexer.Config{
		DataPath:   dataPath,
		Limit:      0, // No limit
		BatchBytes: 1024 * 1024,
		OutputDir:  tempDir,
		Tokenizer:  testTokenizer,
	})

	if _, err := builder.Build(); err != nil {
		log.Fatalf("failed to build test index: %v", err)
	}

	// Load the index
	dict, err := disk.LoadDictionary(indexer.DictionaryPath(tempDir))
	if err != nil {
		log.Fatalf("failed to load dictionary: %v", err)
	}

	testPostingStore, err = disk.NewPostingStore(dict, indexer.PostingsPath(tempDir), 100)
	if err != nil {
		log.Fatalf("failed to open posting store: %v", err)
	}

	// Load document lengths
	testDocLengths, err = index.LoadDocLengths(tempDir)
	if err != nil {
		log.Fatalf("failed to load doc lengths: %v", err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	testPostingStore.Close()
	os.RemoveAll(tempDir)

	os.Exit(code)
}

func TestSearchReturnsErrorOnEmptyQuery(t *testing.T) {
	_, _, _, err := Search(context.Background(), testPostingStore, testTokenizer, nil, "", testDocLengths)
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
	if err.Error() != "query must not be empty" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSearchSingleTermRanksByFrequency(t *testing.T) {
	// Data from testing.tsv:
	// 1001: ranksearch rankengine rankengine (search:1, engine:2)
	// 1002: rankengine ranksearch (search:1, engine:1)
	// 1003: ranksearch ranksearch rankengine (search:2, engine:1)

	_, results, total, err := Search(context.Background(), testPostingStore, testTokenizer, nil, "ranksearch rankengine", testDocLengths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 3 {
		t.Fatalf("expected total results to be 3, got %d", total)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Expected ranking:
	// 1. 1001 (freq 3)
	// 2. 1003 (freq 3)
	// Note: 1001 and 1003 have same total freq, but we check specific values.
	// 3. 1002 (freq 2)

	// Check for existence and correct frequencies
	found := make(map[uint32]int)
	for _, r := range results {
		found[r.DocID] = r.TotalTermFrequency
	}

	if found[1001] != 3 {
		t.Errorf("doc 1001: expected freq 3, got %d", found[1001])
	}
	if found[1003] != 3 {
		t.Errorf("doc 1003: expected freq 3, got %d", found[1003])
	}
	if found[1002] != 2 {
		t.Errorf("doc 1002: expected freq 2, got %d", found[1002])
	}
}

func TestSearchMissingTokenReturnsNil(t *testing.T) {
	// Data: 2001: misssearch missterm
	_, results, total, err := Search(context.Background(), testPostingStore, testTokenizer, nil, "missingtokenxyz", testDocLengths)
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
	// Data: 3000-3010 (11 docs) all have "limitterm"
	_, results, total, err := Search(context.Background(), testPostingStore, testTokenizer, nil, "limitterm", testDocLengths)
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
	// Data:
	// 4001: notcat notdog
	// 4002: notcat notbird
	// 4003: notcat notdog notbird

	// Query: notcat AND NOT notdog
	// Should match 4002 only.
	_, results, total, err := Search(context.Background(), testPostingStore, testTokenizer, nil, "notcat and not notdog", testDocLengths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 1 {
		t.Fatalf("expected total results to be 1, got %d", total)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].DocID != 4002 {
		t.Fatalf("expected 4002 (has cat but not dog), got %v", results[0].DocID)
	}
}

func TestSearchOrPrecedence(t *testing.T) {
	// Data:
	// 5001: orcat ordog
	// 5002: orbird
	// 5003: orcat orbird
	// 5004: ordog

	// Query: orcat AND ordog OR orbird
	// Precedence: (orcat AND ordog) OR orbird
	// Matches:
	// - (orcat AND ordog) -> 5001
	// - orbird -> 5002, 5003
	// Result: 5001, 5002, 5003

	_, results, total, err := Search(context.Background(), testPostingStore, testTokenizer, nil, "orcat and ordog or orbird", testDocLengths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 total matches, got %d", total)
	}

	want := []uint32{5001, 5002, 5003}
	// Sort results by ID for comparison as scores might vary slightly or be tied
	var got []uint32
	for _, r := range results {
		got = append(got, r.DocID)
	}
	slices.Sort(got)
	// want is already sorted

	if !slices.Equal(got, want) {
		t.Fatalf("expected docs %v, got %v", want, got)
	}
}

func TestSearchParenthesesOverridePrecedence(t *testing.T) {
	// Data:
	// 6001: parencat parenbird
	// 6002: parendog parenbird
	// 6003: parenbird

	// Query: (parencat OR parendog) AND parenbird
	// Matches:
	// - parencat OR parendog -> 6001, 6002
	// - AND parenbird -> 6001, 6002
	// 6003 has bird but neither cat nor dog, so excluded.

	_, results, total, err := Search(context.Background(), testPostingStore, testTokenizer, nil, "(parencat or parendog) and parenbird", testDocLengths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 matches, got %d", total)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	got := []uint32{results[0].DocID, results[1].DocID}
	slices.Sort(got)
	want := []uint32{6001, 6002}

	if !slices.Equal(got, want) {
		t.Fatalf("expected docs %v, got %v", want, got)
	}
}

func TestSearchPhraseQueryMatchesContiguousTokens(t *testing.T) {
	// Data:
	// 7001: phrasequick phrasebrown phrasefox
	// 7002: phrasequick phrasefox phrasebrown
	// 7003: phrasequick phrasebrown phrasequick phrasebrown

	// Query: "phrasequick phrasebrown"
	// Matches: 7001 (once), 7003 (twice)
	// 7002 has tokens but not contiguous.

	_, results, total, err := Search(context.Background(), testPostingStore, testTokenizer, nil, "\"phrasequick phrasebrown\"", testDocLengths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 2 {
		t.Fatalf("expected total results to be 2 (ID 7001 and 7003), got %d", total)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// 7003 should be first (freq 2)
	if results[0].DocID != 7003 || results[0].TotalTermFrequency != 2 {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	// Check matches for 7003
	if len(results[0].Matches) < 2 {
		t.Fatalf("expected at least 2 token matches in 7003, got %v", results[0].Matches)
	}
	head := results[0].Matches[0]
	if head.Token != "phrasequick" || head.Frequency != 2 {
		t.Fatalf("unexpected head match for 7003: %+v", head)
	}

	// 7001 should be second (freq 1)
	if results[1].DocID != 7001 || results[1].TotalTermFrequency != 1 {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
}

func TestSearchOnlyNegationsReturnError(t *testing.T) {
	// Data: 8001: negcat
	_, _, _, err := Search(context.Background(), testPostingStore, testTokenizer, nil, "not negcat", testDocLengths)
	if err == nil {
		t.Fatal("expected error for negation-only query, got nil")
	}
	if !strings.Contains(err.Error(), "query must contain at least one positive term") {
		t.Fatalf("unexpected error: %v", err)
	}
}
