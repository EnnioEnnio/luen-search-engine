package data

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempDataset(t *testing.T, rows []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "dataset.tsv")
	if err := os.WriteFile(path, []byte(strings.Join(rows, "\n")), 0o644); err != nil {
		t.Fatalf("write dataset: %v", err)
	}
	return path
}

func TestBatchLoaderRespectsBatchSizeAndLimit(t *testing.T) {
	rows := []string{
		"1\turl1\ttitle1\ttext one",
		"2\turl2\ttitle2\ttext two",
		"3\turl3\ttitle3\ttext three",
	}
	path := writeTempDataset(t, rows)
	loader, err := NewBatchLoader(path, 2, 1)
	if err != nil {
		t.Fatalf("create loader: %v", err)
	}
	t.Cleanup(func() { loader.Close() })

	batch, err := loader.NextBatch()
	if err != nil {
		t.Fatalf("next batch: %v", err)
	}
	if len(batch) != 1 {
		t.Fatalf("expected 1 doc in first batch, got %d", len(batch))
	}

	batch, err = loader.NextBatch()
	if err != nil {
		t.Fatalf("next batch 2: %v", err)
	}
	if len(batch) != 1 {
		t.Fatalf("expected 1 doc in second batch, got %d", len(batch))
	}

	if _, err = loader.NextBatch(); err == nil {
		t.Fatalf("expected EOF after limit, got nil error")
	}

	if loader.DocsRead() != 2 {
		t.Fatalf("expected DocsRead=2, got %d", loader.DocsRead())
	}
}
