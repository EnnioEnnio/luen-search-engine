package index

import (
	"luen-search-engine/internal/data"
	"luen-search-engine/internal/text"
)

type DocID = string
type Token = string
type Position = int

// PostingList stores document frequencies and postings for a single token.
type PostingList struct {
	DocFreq int
	Docs    map[DocID][]Position
}

// InvertedIndex connects tokens to their posting lists.
type InvertedIndex map[Token]*PostingList

// Build constructs an inverted index for the provided documents.
func Build(docs []data.Document, tokenizer text.Tokenizer) InvertedIndex {
	idx := make(InvertedIndex)

	for _, doc := range docs {
		tokens, _ := tokenizer.Tokenize(doc.Title)
		tokensBody, _ := tokenizer.Tokenize(doc.Text)
		tokens = append(tokens, tokensBody...)

		for i, token := range tokens {
			posting, ok := idx[token]
			if !ok {
				positionList := make([]Position, 0)
				positionList = append(positionList, Position(i))
				posting = &PostingList{
					Docs: make(map[DocID][]Position),
				}
				posting.Docs[doc.ID] = positionList
				idx[token] = posting
			}

			positions, seen := posting.Docs[doc.ID]
			positions = append(positions, i)
			posting.Docs[doc.ID] = positions
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
