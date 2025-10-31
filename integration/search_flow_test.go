package integration_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/output"
	"luen-search-engine/internal/search"
	"luen-search-engine/internal/text"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("copy output: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
	return buf.String()
}

func TestEndToEndSearchFlow(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "dataset.tsv")

	rows := []string{
		"doc1\thttps://example.com/1\tFirst Title\tQuick brown fox jumps over the lazy dog",
		"doc2\thttps://example.com/2\tSecond Title\tLazy dogs stay asleep",
		"doc3\thttps://example.com/3\tThird Title\tQuick recipes for busy people",
	}

	if err := os.WriteFile(dataPath, []byte(strings.Join(rows, "\n")), 0o600); err != nil {
		t.Fatalf("write dataset: %v", err)
	}

	tokenizer := text.NewTokenizer()

	dataset, err := data.Load(dataPath, 0)
	if err != nil {
		t.Fatalf("load dataset: %v", err)
	}
	if dataset.Size() != 3 {
		t.Fatalf("expected dataset of size 3, got %d", dataset.Size())
	}

	inverted := index.Build(dataset.Documents, tokenizer)
	if inverted.TokenCount() == 0 {
		t.Fatal("expected tokens in inverted index")
	}

	results, total, err := search.Search(inverted, tokenizer, "quick lazy")
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected a single matching document, got %d", len(results))
	}
	if total != 1 {
		t.Fatalf("expected total result count to be 1, got %d", total)
	}
	if results[0].DocID != "doc1" {
		t.Fatalf("expected doc1 to match, got %s", results[0].DocID)
	}

	outputText := captureStdout(t, func() {
		output.PrintResults(results, dataset, total)
	})

	expectedFragments := []string{
		"Result Count: '1'",
		"https://example.com/1",
		"First Title",
		"quick count: 1",
		"lazy count: 1",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(outputText, fragment) {
			t.Fatalf("expected output to contain %q, got %q", fragment, outputText)
		}
	}
}
