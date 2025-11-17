package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"luen-search-engine/internal/text"
)

func writeDataset(t *testing.T, rows []string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dataset.tsv")
	if err := os.WriteFile(path, []byte(strings.Join(rows, "\n")), 0o644); err != nil {
		t.Fatalf("write dataset: %v", err)
	}
	return path
}

func TestBuilderCreatesOnDiskIndex(t *testing.T) {
	rows := []string{
		"1\turl1\talpha\tbeta", // tokens: alpha, beta
		"2\turl2\tbeta\talpha beta",
		"3\turl3\tgamma\tbeta gamma",
	}
	dataPath := writeDataset(t, rows)
	outputDir := filepath.Join(t.TempDir(), "index")
	builder := NewBuilder(Config{
		DataPath:   dataPath,
		BatchBytes: 32,
		OutputDir:  outputDir,
		Tokenizer:  text.NewTokenizer(),
	})

	manifest, err := builder.Build()
	if err != nil {
		t.Fatalf("build index: %v", err)
	}

	if manifest.DocumentCount != 3 {
		t.Fatalf("expected 3 documents, got %d", manifest.DocumentCount)
	}
	if manifest.TokenCount == 0 {
		t.Fatalf("expected token count > 0")
	}

	if _, err := os.Stat(filepath.Join(outputDir, dictionaryFileName)); err != nil {
		t.Fatalf("dictionary missing: %v", err)
	}
	info, err := os.Stat(filepath.Join(outputDir, postingsFileName))
	if err != nil {
		t.Fatalf("postings missing: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected postings file to be non-empty")
	}
	if _, err := os.Stat(filepath.Join(outputDir, manifestFileName)); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, dictionaryFileName))
	if err != nil {
		t.Fatalf("read dictionary: %v", err)
	}
	contents := string(data)
	if !strings.Contains(contents, "alpha") || !strings.Contains(contents, "beta") {
		t.Fatalf("dictionary missing expected tokens: %s", contents)
	}
}
