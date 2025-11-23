package disk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDictionary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dictionary.tsv")
	content := "alpha\t3\t0\t24\n beta\t1\t24\t8\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write dictionary: %v", err)
	}

	dict, err := LoadDictionary(path)
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}
	if dict.Size() != 2 {
		t.Fatalf("expected 2 entries, got %d", dict.Size())
	}

	entry, ok := dict.Lookup("alpha")
	if !ok {
		t.Fatalf("alpha missing")
	}
	if entry.DocFreq != 3 || entry.Offset != 0 || entry.Length != 24 {
		t.Fatalf("unexpected alpha entry: %+v", entry)
	}

	if _, ok := dict.Lookup("missing"); ok {
		t.Fatalf("expected missing token to fail lookup")
	}
}
