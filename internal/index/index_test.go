package index

import (
	"testing"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/text"
)

func TestBuildCreatesPostingLists(t *testing.T) {
	docs := []data.Document{
		{
			ID:    123,
			Title: "First Document",
			Text:  "Go search engine search",
		},
		{
			ID:    456,
			Title: "Second Document",
			Text:  "Search engine basics",
		},
	}

	tokenizer := text.NewTokenizer()
	idx := Build(docs, tokenizer)

	if got := idx.TokenCount(); got == 0 {
		t.Fatalf("expected tokens to be indexed")
	}

	searchPosting, ok := idx["search"]
	if !ok {
		t.Fatalf("expected posting list for token 'search'")
	}
	if searchPosting.DocFreq != 2 {
		t.Fatalf("expected doc freq 2 for 'search', got %d", searchPosting.DocFreq)
	}
	if got := searchPosting.Docs[123]; len(got) != 2 {
		t.Fatalf("expected docA frequency 2 for 'search', got %d", len(got))
	}
	if got := searchPosting.Docs[456]; len(got) != 1 {
		t.Fatalf("expected docB frequency 1 for 'search', got %d", len(got))
	}

	enginePosting, ok := idx["engine"]
	if !ok {
		t.Fatalf("expected posting list for token 'engine'")
	}
	if enginePosting.DocFreq != 2 {
		t.Fatalf("expected doc freq 2 for 'engine', got %d", enginePosting.DocFreq)
	}
}
