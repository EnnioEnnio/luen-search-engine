package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"hpi-search-engine/internal/data"
	"hpi-search-engine/internal/search"
)

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
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

func TestPrintResultsNoResults(t *testing.T) {
	ds := &data.Dataset{
		ByID: make(map[string]data.Document),
	}

	out := captureOutput(t, func() {
		PrintResults(nil, ds, 0)
	})

	if !strings.Contains(out, "No results found") {
		t.Fatalf("expected no results message, got %q", out)
	}
}

func TestPrintResultsWithMatches(t *testing.T) {
	ds := &data.Dataset{
		Documents: []data.Document{
			{ID: "doc1", URL: "https://example.com", Title: "Example Title"},
		},
		ByID: map[string]data.Document{
			"doc1": {ID: "doc1", URL: "https://example.com", Title: "Example Title"},
		},
	}

	results := []search.Result{
		{
			DocID: "doc1",
			Matches: []search.Match{
				{Token: "search", Frequency: 2},
				{Token: "engine", Frequency: 1},
			},
		},
	}

	out := captureOutput(t, func() {
		PrintResults(results, ds, len(results))
	})

	expectedFragments := []string{
		"Result Count: '1'",
		"Results:",
		"https://example.com",
		"Example Title",
		"search count: 2",
		"engine count: 1",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(out, fragment) {
			t.Fatalf("expected output to contain %q, got %q", fragment, out)
		}
	}
}
