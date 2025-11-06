package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadInBatchesReadsDocumentsAndLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs.tsv")

	lines := []string{
		"doc1\thttps://example.com/1\tFirst Title\tFirst text segment",
		"doc2\thttps://example.com/2\tSecond Title\tText with\textra\tsegments",
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write dataset: %v", err)
	}

	batchCount := 0
	totalDocs := 0
	ds, err := LoadInBatches(path, 10, 0, func(batch []Document) error {
		batchCount++
		totalDocs += len(batch)
		return nil
	})
	if err != nil {
		t.Fatalf("LoadInBatches returned error: %v", err)
	}

	if ds == nil {
		t.Fatal("expected dataset, got nil")
	}
	if got := ds.Size(); got != 2 {
		t.Fatalf("expected 2 documents, got %d", got)
	}
	if totalDocs != 2 {
		t.Fatalf("expected 2 documents processed in batches, got %d", totalDocs)
	}

	doc2, ok := ds.ByID["doc2"]
	if !ok {
		t.Fatalf("expected doc2 to be present in lookup")
	}
	// Note: MetadataDoc doesn't have Text field, only ID, URL, Title
	if doc2.Title != "Second Title" {
		t.Fatalf("expected title %q, got %q", "Second Title", doc2.Title)
	}

	// Test with limit
	batchCount = 0
	totalDocs = 0
	limited, err := LoadInBatches(path, 10, 1, func(batch []Document) error {
		batchCount++
		totalDocs += len(batch)
		return nil
	})
	if err != nil {
		t.Fatalf("LoadInBatches with limit returned error: %v", err)
	}
	if got := limited.Size(); got != 1 {
		t.Fatalf("expected 1 document with limit, got %d", got)
	}
	if totalDocs != 1 {
		t.Fatalf("expected 1 document processed with limit, got %d", totalDocs)
	}
}

func TestLoadInBatchesProcessesPartialFinalBatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs.tsv")

	// Create 1000 documents
	lines := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		docID := fmt.Sprintf("doc%04d", i)
		lines[i] = docID + "\thttps://example.com/" + docID + "\tTitle " + docID + "\tText for " + docID
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write dataset: %v", err)
	}

	// Test with batchSize=99, limit=1000
	// Should produce: 10 batches of 99 + 1 batch of 10 = 11 batches total
	batchSizes := []int{}
	batchCount := 0
	totalDocs := 0

	ds, err := LoadInBatches(path, 99, 1000, func(batch []Document) error {
		batchCount++
		batchSizes = append(batchSizes, len(batch))
		totalDocs += len(batch)
		return nil
	})
	if err != nil {
		t.Fatalf("LoadInBatches returned error: %v", err)
	}

	if batchCount != 11 {
		t.Fatalf("expected 11 batches, got %d", batchCount)
	}

	if totalDocs != 1000 {
		t.Fatalf("expected 1000 documents processed, got %d", totalDocs)
	}

	if ds.Size() != 1000 {
		t.Fatalf("expected 1000 documents in dataset, got %d", ds.Size())
	}

	// Verify first 10 batches have 99 documents
	for i := 0; i < 10; i++ {
		if batchSizes[i] != 99 {
			t.Errorf("batch %d: expected 99 documents, got %d", i+1, batchSizes[i])
		}
	}

	// Verify final batch has 10 documents (1000 - 10*99 = 10)
	if batchSizes[10] != 10 {
		t.Errorf("final batch: expected 10 documents, got %d", batchSizes[10])
	}
}

func TestLoadInBatchesWithEvenBatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs.tsv")

	// Create 1000 documents
	lines := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		docID := fmt.Sprintf("doc%04d", i)
		lines[i] = docID + "\thttps://example.com/" + docID + "\tTitle " + docID + "\tText for " + docID
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write dataset: %v", err)
	}

	// Test with batchSize=100, limit=1000
	// Should produce exactly 10 batches of 100 each
	batchSizes := []int{}
	batchCount := 0

	_, err := LoadInBatches(path, 100, 1000, func(batch []Document) error {
		batchCount++
		batchSizes = append(batchSizes, len(batch))
		return nil
	})
	if err != nil {
		t.Fatalf("LoadInBatches returned error: %v", err)
	}

	if batchCount != 10 {
		t.Fatalf("expected 10 batches, got %d", batchCount)
	}

	// Verify all batches have exactly 100 documents
	for i, size := range batchSizes {
		if size != 100 {
			t.Errorf("batch %d: expected 100 documents, got %d", i+1, size)
		}
	}
}

func TestLoadInBatchesWithNegativeLimit(t *testing.T) {
	if _, err := LoadInBatches("ignored", 10, -1, nil); err == nil {
		t.Fatal("expected error for negative limit, got nil")
	}
}

func TestLoadInBatchesWithInvalidBatchSize(t *testing.T) {
	if _, err := LoadInBatches("ignored", 0, 0, nil); err == nil {
		t.Fatal("expected error for zero batch size, got nil")
	}

	if _, err := LoadInBatches("ignored", -1, 0, nil); err == nil {
		t.Fatal("expected error for negative batch size, got nil")
	}
}

func TestDatasetSizeOnNil(t *testing.T) {
	var ds *Dataset
	if ds.Size() != 0 {
		t.Fatalf("nil dataset should report size 0")
	}
}
