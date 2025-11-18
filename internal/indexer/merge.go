package indexer

import (
	"bufio"
	"bytes"
	"container/heap"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"luen-search-engine/internal/model"
)

// postingHeap implements heap.Interface over partialReader streams sorted by token.
type postingHeap []*partialReader

func (h postingHeap) Len() int { return len(h) }
func (h postingHeap) Less(i, j int) bool {
	ci := h[i].current
	cj := h[j].current
	if ci.Token == cj.Token {
		return h[i].id < h[j].id
	}
	return ci.Token < cj.Token
}
func (h postingHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *postingHeap) Push(x interface{}) {
	*h = append(*h, x.(*partialReader))
}
func (h *postingHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}
func (h postingHeap) peek() *partialReader {
	if len(h) == 0 {
		return nil
	}
	return h[0]
}

// mergePartials performs a k-way merge of gob partials and writes the final postings/dictionary files.
func mergePartials(paths []string, outputDir string, keepPartials bool) (int, error) {
	postingsPath := filepath.Join(outputDir, postingsFileName)
	dictionaryPath := filepath.Join(outputDir, dictionaryFileName)

	postingsFile, err := os.Create(postingsPath)
	if err != nil {
		return 0, fmt.Errorf("create postings file: %w", err)
	}
	defer postingsFile.Close()

	dictWriter, err := newDictionaryWriter(dictionaryPath)
	if err != nil {
		return 0, err
	}
	defer dictWriter.Close()

	h := make(postingHeap, 0, len(paths))
	readers := make([]*partialReader, 0, len(paths))
	for i, path := range paths {
		reader, err := newPartialReader(path, i)
		if err != nil {
			return 0, err
		}
		readers = append(readers, reader)
		if err := reader.advance(); err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			return 0, err
		}
		h = append(h, reader)
	}
	if len(h) == 0 {
		return 0, fmt.Errorf("no partial postings to merge")
	}
	heap.Init(&h)

	var tokenCount int
	for h.Len() > 0 {
		cursor := heap.Pop(&h).(*partialReader)
		token := cursor.current.Token
		combinedDocs := append([]diskDocPosting(nil), cursor.current.Docs...)

		for {
			next := postingHeap(h).peek()
			if next == nil || next.current.Token != token {
				break
			}
			other := heap.Pop(&h).(*partialReader)
			combinedDocs = append(combinedDocs, other.current.Docs...)
			if err := other.advance(); err != nil {
				if errors.Is(err, io.EOF) {
					other.Close()
					continue
				}
				return 0, err
			}
			heap.Push(&h, other)
		}

		normalized := normalizeDocs(combinedDocs)
		posting := diskPosting{Token: token, DocFreq: len(normalized), Docs: normalized}
		offset, length, err := writePosting(postingsFile, posting)
		if err != nil {
			return 0, err
		}
		if err := dictWriter.Write(dictionaryEntry{Token: token, DocFreq: posting.DocFreq, Offset: offset, Length: length}); err != nil {
			return 0, err
		}
		tokenCount++

		if err := cursor.advance(); err != nil {
			if errors.Is(err, io.EOF) {
				cursor.Close()
			} else {
				return 0, err
			}
		} else {
			heap.Push(&h, cursor)
		}
	}

	if !keepPartials {
		for _, path := range paths {
			_ = os.Remove(path)
		}
	}

	for _, reader := range readers {
		reader.Close()
	}

	return tokenCount, nil
}

// normalizeDocs sorts postings for a token and merges duplicate doc entries.
func normalizeDocs(docs []diskDocPosting) []diskDocPosting {
	if len(docs) <= 1 {
		return docs
	}
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].DocID == docs[j].DocID {
			return len(docs[i].Positions) < len(docs[j].Positions)
		}
		return docs[i].DocID < docs[j].DocID
	})

	compact := docs[:0]
	var lastDocID model.DocID = ^model.DocID(0)
	for _, doc := range docs {
		if len(doc.Positions) == 0 {
			continue
		}
		if doc.DocID == lastDocID {
			compact[len(compact)-1].Positions = append(compact[len(compact)-1].Positions, doc.Positions...)
			continue
		}
		compact = append(compact, doc)
		lastDocID = doc.DocID
	}

	for i := range compact {
		sort.Ints(compact[i].Positions)
	}

	return compact
}

// dictionaryEntry represents a single line inside dictionary.tsv.
type dictionaryEntry struct {
	Token   string
	DocFreq int
	Offset  int64
	Length  int64
}

// dictionaryWriter streams dictionary entries to disk in TSV format.
type dictionaryWriter struct {
	file   *os.File
	writer *bufio.Writer
}

// newDictionaryWriter creates a buffered writer for dictionary.tsv.
func newDictionaryWriter(path string) (*dictionaryWriter, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create dictionary file: %w", err)
	}
	return &dictionaryWriter{file: file, writer: bufio.NewWriter(file)}, nil
}

// Write appends a single dictionary entry to the file.
func (w *dictionaryWriter) Write(entry dictionaryEntry) error {
	if w == nil {
		return fmt.Errorf("dictionary writer is nil")
	}
	if _, err := fmt.Fprintf(w.writer, "%s\t%d\t%d\t%d\n", entry.Token, entry.DocFreq, entry.Offset, entry.Length); err != nil {
		return fmt.Errorf("write dictionary entry: %w", err)
	}
	return nil
}

// Close flushes any buffered dictionary data and closes the file handle.
func (w *dictionaryWriter) Close() error {
	if w == nil {
		return nil
	}
	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.file.Close()
}

// writePosting persists an encoded posting list and returns its file offset/length.
func writePosting(file *os.File, posting diskPosting) (int64, int64, error) {
	offset, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, 0, fmt.Errorf("seek postings file: %w", err)
	}

	buf := bytes.NewBuffer(make([]byte, 0, len(posting.Docs)*32))
	if err := binary.Write(buf, binary.LittleEndian, uint32(len(posting.Docs))); err != nil {
		return 0, 0, fmt.Errorf("write docfreq: %w", err)
	}
	for _, doc := range posting.Docs {
		if err := binary.Write(buf, binary.LittleEndian, uint32(doc.DocID)); err != nil {
			return 0, 0, fmt.Errorf("write doc id: %w", err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(len(doc.Positions))); err != nil {
			return 0, 0, fmt.Errorf("write position count: %w", err)
		}
		for _, pos := range doc.Positions {
			if pos < 0 {
				return 0, 0, fmt.Errorf("negative position for token %s", posting.Token)
			}
			if err := binary.Write(buf, binary.LittleEndian, uint32(pos)); err != nil {
				return 0, 0, fmt.Errorf("write position: %w", err)
			}
		}
	}

	n, err := file.Write(buf.Bytes())
	if err != nil {
		return 0, 0, fmt.Errorf("write postings: %w", err)
	}

	return offset, int64(n), nil
}
