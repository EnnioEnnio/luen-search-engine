package search

import (
	"fmt"
	"sort"

	"hpi-search-engine/internal/index"
	"hpi-search-engine/internal/text"
)

// Match keeps track of how often a token occurred in a document.
type Match struct {
	Token     string
	Frequency int
}

// Result represents the aggregated scores for a matching document.
type Result struct {
	DocID              string
	TotalTermFrequency int
	Matches            []Match
}

// Search finds documents that contain all query tokens and orders them by frequency.
func Search(idx index.InvertedIndex, tokenizer text.Tokenizer, query string) ([]Result, error) {
	if query == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	tokens := tokenizer.Tokenize(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	docLists := make([]map[string]Match, 0, len(tokens))
	for _, token := range tokens {
		posting, ok := idx[token]
		if !ok {
			return nil, nil
		}

		docs := make(map[string]Match, len(posting.Docs))
		for docID, freq := range posting.Docs {
			docs[docID] = Match{Token: token, Frequency: freq}
		}
		docLists = append(docLists, docs)
	}

	if len(docLists) == 0 {
		return nil, nil
	}

	results := make([]Result, 0)

	for docID, headMatch := range docLists[0] {
		total := headMatch.Frequency
		matches := []Match{headMatch}
		missing := false

		for _, docs := range docLists[1:] {
			match, ok := docs[docID]
			if !ok {
				missing = true
				break
			}
			total += match.Frequency
			matches = append(matches, match)
		}

		if missing {
			continue
		}

		results = append(results, Result{
			DocID:              docID,
			TotalTermFrequency: total,
			Matches:            matches,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].TotalTermFrequency == results[j].TotalTermFrequency {
			return results[i].DocID < results[j].DocID
		}
		return results[i].TotalTermFrequency > results[j].TotalTermFrequency
	})

	if len(results) > 10 {
		results = results[:10]
	}

	return results, nil
}
