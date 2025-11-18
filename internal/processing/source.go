package processing

import "luen-search-engine/internal/index"

// MemorySource wraps an in-memory inverted index so the parser can evaluate queries without disk IO.
type MemorySource struct {
	idx index.InvertedIndex
}

var _ index.PostingSource = (*MemorySource)(nil)

// NewMemorySource returns a PostingSource backed by the provided inverted index.
func NewMemorySource(idx index.InvertedIndex) *MemorySource {
	return &MemorySource{idx: idx}
}

// Lookup implements PostingSource by returning the existing posting list for the token.
func (m *MemorySource) Lookup(token index.Token) (*index.PostingList, error) {
	if m == nil || m.idx == nil {
		return nil, nil
	}
	posting, ok := m.idx[token]
	if !ok {
		return nil, nil
	}
	return posting, nil
}
