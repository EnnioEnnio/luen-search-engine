package data

import (
	"bufio"
	"encoding/csv"
	"errors"
	"io"
	"os"
)

// Document represents a single record from the MS MARCO dataset.
type Document struct {
	ID    string
	URL   string
	Title string
	Text  string
}

type MetadataDoc struct {
	ID    string
	URL   string
	Title string
}

// Dataset keeps the loaded documents together with a direct lookup table.
type Dataset struct {
	ByID map[string]MetadataDoc
}

// LoadInBatches processes a file in batches, calling processBatch for each batch.
// This allows incremental processing of large datasets that don't fit in memory.
// Unlike the previous approach, this builds the full index incrementally
// while keeping only lightweight metadata (without document text) in the dataset.
func LoadInBatches(path string, batchSize int, totalLimit int, processBatch func(batch []Document) error) (*Dataset, error) {
	if batchSize <= 0 {
		return nil, errors.New("batchSize must be positive")
	}
	if totalLimit < 0 {
		return nil, errors.New("totalLimit must be zero or positive")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	lookup := make(map[string]MetadataDoc)
	currentBatch := make([]Document, 0, batchSize)
	totalLoaded := 0

	for {
		if totalLimit != 0 && totalLoaded >= totalLimit {
			// Process final batch if it has any documents
			if len(currentBatch) > 0 {
				if err := processBatch(currentBatch); err != nil {
					return nil, err
				}
			}
			break
		}

		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			// Process final batch if it has any documents
			if len(currentBatch) > 0 {
				if err := processBatch(currentBatch); err != nil {
					return nil, err
				}
			}
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < 4 {
			continue
		}

		textField := record[3]
		if len(record) > 4 {
			// When additional tab characters appear in the text, join them back together.
			for i := 4; i < len(record); i++ {
				textField += "\t" + record[i]
			}
		}

		doc := Document{
			ID:    record[0],
			URL:   record[1],
			Title: record[2],
			Text:  textField,
		}

		metadataDoc := MetadataDoc{
			ID:    record[0],
			URL:   record[1],
			Title: record[2],
		}

		currentBatch = append(currentBatch, doc)
		lookup[metadataDoc.ID] = metadataDoc
		totalLoaded++

		// Process batch when it reaches the batch size
		if len(currentBatch) >= batchSize {
			if err := processBatch(currentBatch); err != nil {
				return nil, err
			}
			currentBatch = make([]Document, 0, batchSize)
		}
	}

	return &Dataset{
		ByID: lookup,
	}, nil
}

// Size returns the number of documents stored in the dataset.
func (d *Dataset) Size() int {
	if d == nil {
		return 0
	}
	return len(d.ByID)
}
