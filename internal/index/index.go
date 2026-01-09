package index

import (
	"luen-search-engine/internal/data"
	"luen-search-engine/internal/model"
	"luen-search-engine/internal/text"
)

type DocID = model.DocID
type Token = model.Token
type Position = int
type DocLength = model.FieldDocLengths

// PostingList stores document frequencies and positions within each document for a single token.
type PostingList struct {
	DocFreq int
	Docs    map[DocID][]Position
}

// InvertedIndex connects tokens to their posting lists.
type InvertedIndex map[Token]*PostingList

// BuildResult holds the constructed inverted index and document lengths.
type BuildResult struct {
	Index      InvertedIndex
	DocLengths map[DocID]model.FieldDocLengths
}

// Build constructs an inverted index for the provided documents.
func Build(docs []data.Document, tokenizer *text.Tokenizer) BuildResult {
	const approxVocab = 100_000 // 1000 documents have roughly around 70.000 unique token according to `make dev`
	idx := make(InvertedIndex, approxVocab)
	docLengths := make(map[DocID]DocLength, len(docs))

	for _, doc := range docs {
		tokensTitle := tokenizer.Tokenize(doc.Title)
		tokensBody := tokenizer.Tokenize(doc.Text)
		tokens := make([]Token, 0, len(tokensTitle)+len(tokensBody))
		tokens = append(tokens, tokensTitle...)
		tokens = append(tokens, tokensBody...)

		for i, token := range tokens {
			posting, ok := idx[token]
			if !ok {
				posting = &PostingList{
					Docs: make(map[DocID][]Position, 1),
				}
				idx[token] = posting
			}

			positions, seen := posting.Docs[doc.ID]
			posting.Docs[doc.ID] = append(positions, i)
			if !seen {
				posting.DocFreq++
			}
		}
		docLengths[doc.ID] = DocLength{
			TitleLength: uint32(len(tokensTitle)),
			BodyLength:  uint32(len(tokensBody)),
		}
	}

	return BuildResult{
		Index:      idx,
		DocLengths: docLengths,
	}
}

// TokenCount returns the number of unique tokens held in the index.
func (ii InvertedIndex) TokenCount() int {
	return len(ii)
}
