package data

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Document represents a single record from the MS MARCO dataset.
type Document struct {
	ID    uint32
	URL   string
	Title string
	Text  string
}

// Dataset keeps the loaded documents together with a direct lookup table.
type Dataset struct {
	Documents []Document
	ByID      map[uint32]Document
}

// Load reads up to `limit` rows from a tab-separated file and returns a Dataset.
// The file is expected to be ordered as: docId, url, title, text – without a header.
func Load(path string, limit int) (*Dataset, error) {
	if limit < 0 {
		return nil, errors.New("limit must be zero or positive")
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

	docs := make([]Document, 0, limit)
	lookup := make(map[uint32]Document)

	for {
		if limit != 0 && len(docs) >= limit {
			break
		}

		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
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

		stringID := record[0]
		value, err := strconv.ParseUint(stringID, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid docID %q: %w", stringID, err)
		}

		doc := Document{
			ID:    uint32(value),
			URL:   record[1],
			Title: record[2],
			Text:  textField,
		}

		docs = append(docs, doc)
		lookup[doc.ID] = doc
	}

	return &Dataset{
		Documents: docs,
		ByID:      lookup,
	}, nil
}

// Size returns the number of documents stored in the dataset.
func (d *Dataset) Size() int {
	if d == nil {
		return 0
	}
	return len(d.Documents)
}
