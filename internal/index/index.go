package index

import (
	"luen-search-engine/internal/data"
	"luen-search-engine/internal/text"
)

// PostingList stores document frequencies and postings for a single token.
type PostingList struct {
	DocFreq int
	Docs    map[string]int
}

// InvertedIndex connects tokens to their posting lists.
type InvertedIndex map[string]*PostingList

// Build constructs an inverted index for the provided documents.
func Build(docs []data.Document, tokenizer text.Tokenizer) InvertedIndex {
	idx := make(InvertedIndex)

	for _, doc := range docs {
		tokens, _ := tokenizer.Tokenize(doc.Title)
		tokensBody, _ := tokenizer.Tokenize(doc.Text)
		tokens = append(tokens, tokensBody...)

		for _, token := range tokens {
			posting, ok := idx[token]
			if !ok {
				posting = &PostingList{
					Docs: make(map[string]int),
				}
				idx[token] = posting
			}

			count, seen := posting.Docs[doc.ID]
			posting.Docs[doc.ID] = count + 1
			if !seen {
				posting.DocFreq++
			}
		}
	}

	return idx
}

// TokenCount returns the number of unique tokens held in the index.
func (ii InvertedIndex) TokenCount() int {
	return len(ii)
}
