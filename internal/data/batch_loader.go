package data

import (
	"bufio"
	"encoding/csv"
	"errors"
	"io"
	"os"
)

const defaultBatchSizeBytes = 64 * 1024 * 1024

// BatchLoader streams documents from disk in bounded batches to keep memory usage low.
type BatchLoader struct {
	file       *os.File
	reader     *csv.Reader
	limit      int
	readDocs   int
	batchBytes int64
	done       bool
}

// NewBatchLoader returns a loader that reads records from the TSV at path.
func NewBatchLoader(path string, limit int, batchBytes int64) (*BatchLoader, error) {
	if limit < 0 {
		return nil, errors.New("limit must be zero or positive")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(bufio.NewReader(file))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	if batchBytes <= 0 {
		batchBytes = defaultBatchSizeBytes
	}

	return &BatchLoader{
		file:       file,
		reader:     reader,
		limit:      limit,
		batchBytes: batchBytes,
	}, nil
}

// NextBatch loads the next slice of documents whose combined payload roughly matches batchBytes.
func (b *BatchLoader) NextBatch() ([]Document, error) {
	if b == nil || b.done {
		return nil, io.EOF
	}

	batch := make([]Document, 0, 10_000) // to keep memory under 1GB we process 64 MiB, which is < 10_000 records
	var payload int64

	for {
		if b.limit > 0 && b.readDocs >= b.limit {
			b.done = true
			break
		}

		record, err := b.reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				b.done = true
				break
			}
			return nil, err
		}

		doc, err := parseRecord(record)
		if err != nil {
			if errors.Is(err, ErrIncompleteRecord) {
				continue
			}
			return nil, err
		}

		batch = append(batch, doc)
		b.readDocs++
		payload += approxDocumentBytes(doc)

		if payload >= b.batchBytes && len(batch) > 0 {
			break
		}
	}

	if len(batch) == 0 {
		return nil, io.EOF
	}

	return batch, nil
}

// DocsRead returns the total number of documents emitted so far.
func (b *BatchLoader) DocsRead() int {
	if b == nil {
		return 0
	}
	return b.readDocs
}

// Close releases the underlying file handle.
func (b *BatchLoader) Close() error {
	if b == nil || b.file == nil {
		return nil
	}
	return b.file.Close()
}

func approxDocumentBytes(doc Document) int64 {
	// Roughly account for textual payload plus minimal struct overhead.
	return int64(len(doc.URL)+len(doc.Title)+len(doc.Text)) + 16
}
