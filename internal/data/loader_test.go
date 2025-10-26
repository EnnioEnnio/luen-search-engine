package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReadsDocumentsAndLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs.tsv")

	lines := []string{
		"doc1\thttps://example.com/1\tFirst Title\tFirst text segment",
		"doc2\thttps://example.com/2\tSecond Title\tText with\textra\tsegments",
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write dataset: %v", err)
	}

	ds, err := Load(path, 0)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if ds == nil {
		t.Fatal("expected dataset, got nil")
	}
	if got := ds.Size(); got != 2 {
		t.Fatalf("expected 2 documents, got %d", got)
	}

	doc2, ok := ds.ByID["doc2"]
	if !ok {
		t.Fatalf("expected doc2 to be present in lookup")
	}
	expectedText := "Text with\textra\tsegments"
	if doc2.Text != expectedText {
		t.Fatalf("expected merged text %q, got %q", expectedText, doc2.Text)
	}

	limited, err := Load(path, 1)
	if err != nil {
		t.Fatalf("Load with limit returned error: %v", err)
	}
	if got := limited.Size(); got != 1 {
		t.Fatalf("expected 1 document with limit, got %d", got)
	}
}

func TestLoadWithNegativeLimit(t *testing.T) {
	if _, err := Load("ignored", -1); err == nil {
		t.Fatal("expected error for negative limit, got nil")
	}
}

func TestDatasetSizeOnNil(t *testing.T) {
	var ds *Dataset
	if ds.Size() != 0 {
		t.Fatalf("nil dataset should report size 0")
	}
}
