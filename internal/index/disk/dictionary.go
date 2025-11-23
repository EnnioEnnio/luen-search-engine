package disk

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Entry represents a single token row from dictionary.tsv.
type Entry struct {
	Token   string
	DocFreq int
	Offset  int64
	Length  int64
}

// Dictionary holds token metadata required to seek into postings.bin.
type Dictionary struct {
	entries map[string]Entry
}

// LoadDictionary parses the TSV file emitted by the indexer.
func LoadDictionary(path string) (*Dictionary, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open dictionary: %w", err)
	}
	defer file.Close()

	entries := make(map[string]Entry)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) != 4 {
			continue
		}
		token := parts[0]
		docFreq, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("parse docfreq for %s: %w", token, err)
		}
		offset, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse offset for %s: %w", token, err)
		}
		length, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse length for %s: %w", token, err)
		}
		entries[token] = Entry{Token: token, DocFreq: docFreq, Offset: offset, Length: length}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan dictionary: %w", err)
	}

	return &Dictionary{entries: entries}, nil
}

// Lookup returns the dictionary entry for a token.
func (d *Dictionary) Lookup(token string) (Entry, bool) {
	if d == nil {
		return Entry{}, false
	}
	entry, ok := d.entries[token]
	return entry, ok
}

// Size returns the number of tokens tracked inside the dictionary.
func (d *Dictionary) Size() int {
	if d == nil {
		return 0
	}
	return len(d.entries)
}
