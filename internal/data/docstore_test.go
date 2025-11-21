package data

import "testing"

func TestDocumentStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewDocumentStoreWriter(dir)
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	docs := []Document{
		{ID: 1, URL: "https://a", Title: "Doc A", Text: "alpha beta"},
		{ID: 2, URL: "https://b", Title: "Doc B", Text: "beta gamma"},
	}
	for _, doc := range docs {
		if err := writer.Append(doc); err != nil {
			writer.Close()
			t.Fatalf("append doc %d: %v", doc.ID, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	store, err := OpenDocumentStore(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	for _, want := range docs {
		got, ok := store.Lookup(want.ID)
		if !ok {
			t.Fatalf("doc %d missing", want.ID)
		}
		if got != want {
			t.Fatalf("doc mismatch: got %+v, want %+v", got, want)
		}
	}

	if _, ok := store.Lookup(999); ok {
		t.Fatalf("expected missing doc lookup to return false")
	}
}
