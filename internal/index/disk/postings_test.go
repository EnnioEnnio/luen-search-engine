package disk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"luen-search-engine/internal/index"
)

func TestPostingStoreLookupAndCache(t *testing.T) {
	dir := t.TempDir()
	postingsPath := filepath.Join(dir, "postings.bin")
	postingData := encodeTestPosting(t, []testPostingDoc{
		{docID: 1, positions: []uint32{1, 4}},
		{docID: 3, positions: []uint32{2}},
	})
	if err := os.WriteFile(postingsPath, postingData, 0o644); err != nil {
		t.Fatalf("write postings: %v", err)
	}

	path := filepath.Join(dir, "dictionary.tsv")
	line := []byte(fmt.Sprintf("token\t2\t0\t%d\n", len(postingData)))
	if err := os.WriteFile(path, line, 0o644); err != nil {
		t.Fatalf("write dictionary: %v", err)
	}

	dict, err := LoadDictionary(path)
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}
	store, err := NewPostingStore(dict, postingsPath, 8)
	if err != nil {
		t.Fatalf("new posting store: %v", err)
	}
	defer store.Close()

	posting, err := store.Lookup("token")
	if err != nil {
		t.Fatalf("lookup token: %v", err)
	}
	if posting == nil {
		t.Fatalf("expected posting, got nil")
	}
	if posting.DocFreq != 2 {
		t.Fatalf("unexpected docfreq: %d", posting.DocFreq)
	}
	if len(posting.Docs) != 2 {
		t.Fatalf("unexpected docs len: %d", len(posting.Docs))
	}
	positions := posting.Docs[index.DocID(1)]
	if len(positions) != 2 || positions[0] != 1 || positions[1] != 4 {
		t.Fatalf("unexpected positions for doc1: %v", positions)
	}

	// Ensure cache hit doesn't error and returns the same pointer.
	posting2, err := store.Lookup("token")
	if err != nil {
		t.Fatalf("lookup token cached: %v", err)
	}
	if posting != posting2 {
		t.Fatalf("expected cached posting reuse")
	}

	if posting, err := store.Lookup("missing"); err != nil || posting != nil {
		t.Fatalf("expected nil posting for missing token, got %v err=%v", posting, err)
	}
}

type testPostingDoc struct {
	docID     uint32
	positions []uint32
}

func encodeTestPosting(t *testing.T, docs []testPostingDoc) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(docs))); err != nil {
		t.Fatalf("write doc count: %v", err)
	}
	for _, doc := range docs {
		if err := binary.Write(buf, binary.LittleEndian, doc.docID); err != nil {
			t.Fatalf("write doc id: %v", err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(len(doc.positions))); err != nil {
			t.Fatalf("write pos count: %v", err)
		}
		for _, pos := range doc.positions {
			if err := binary.Write(buf, binary.LittleEndian, pos); err != nil {
				t.Fatalf("write pos: %v", err)
			}
		}
	}
	return buf.Bytes()
}
