package indexer

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"

	"luen-search-engine/internal/index"
	"luen-search-engine/internal/model"
)

type DocID = model.DocID

// diskDocPosting stores the serialized postings per document on disk.
type diskDocPosting struct {
	DocID     DocID
	Positions []int
}

// diskPosting stores every token entry we spill to disk.
type diskPosting struct {
	Token   string
	DocFreq int
	Docs    []diskDocPosting
}

// spillPartialIndex encodes an in-memory inverted index into a sorted gob file on disk.
func spillPartialIndex(path string, idx index.InvertedIndex) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create partial index: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	encoder := gob.NewEncoder(writer)
	tokens := make([]string, 0, idx.TokenCount())
	for token := range idx {
		tokens = append(tokens, token)
	}
	sort.Strings(tokens)

	for _, token := range tokens {
		posting := idx[token]
		docIDs := make([]DocID, 0, len(posting.Docs))
		// TODO: build index like this in the first place to speed things up. i.e. remove map
		for docID := range posting.Docs {
			docIDs = append(docIDs, docID)
		}
		sort.Slice(docIDs, func(i, j int) bool { return docIDs[i] < docIDs[j] })

		docs := make([]diskDocPosting, 0, len(docIDs))
		for _, docID := range docIDs {
			positions := append([]int(nil), posting.Docs[docID]...)
			docs = append(docs, diskDocPosting{DocID: docID, Positions: positions})
		}

		entry := diskPosting{Token: token, DocFreq: len(docs), Docs: docs}
		if err := encoder.Encode(&entry); err != nil {
			return fmt.Errorf("encode partial posting for %q: %w", token, err)
		}
	}

	return writer.Flush()
}

// partialReader streams postings from a spilled partial index one entry at a time.
type partialReader struct {
	id      int
	path    string
	file    *os.File
	decoder *gob.Decoder
	current *diskPosting
	done    bool
}

// newPartialReader opens a gob-encoded partial index for streaming.
func newPartialReader(path string, id int) (*partialReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open partial %s: %w", path, err)
	}

	return &partialReader{
		id:      id,
		path:    path,
		file:    file,
		decoder: gob.NewDecoder(bufio.NewReader(file)),
	}, nil
}

// advance decodes the next posting entry from the partial index stream.
func (p *partialReader) advance() error {
	if p.done {
		return io.EOF
	}

	var entry diskPosting
	if err := p.decoder.Decode(&entry); err != nil {
		if errors.Is(err, io.EOF) {
			p.done = true
			p.current = nil
			return io.EOF
		}
		return fmt.Errorf("decode partial %s: %w", p.path, err)
	}

	p.current = &entry
	return nil
}

// Close releases the file handle associated with the partial reader.
func (p *partialReader) Close() error {
	if p == nil || p.file == nil {
		return nil
	}
	err := p.file.Close()
	p.file = nil
	return err
}
