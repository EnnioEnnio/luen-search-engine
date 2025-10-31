package search

import (
	"fmt"
	"sort"
	"strings"

	"luen-search-engine/internal/index"
	"luen-search-engine/internal/text"
)

// Match keeps track of how often a token occurred in a document.
type Match struct {
	Token     string
	Frequency int
}

// Result represents the aggregated scores for a matching document.
type Result struct {
	DocID              string
	TotalTermFrequency int 		// used for ranking only
	Matches            []Match
}

// PreprocessQuery returns a List of maps(key: docID, value: Match), for each token one list
func PreprocessQuery(idx index.InvertedIndex, tokenizer text.Tokenizer, query string) ([]map[string]Match){
	tokens := tokenizer.Tokenize(query)
	docLists := make([]map[string]Match, 0, len(tokens))

	if len(tokens) == 0 {
		return docLists
	}

	for _, token := range tokens {
		posting, ok := idx[token]
		if !ok {
			continue
		}

		docs := make(map[string]Match, len(posting.Docs))
		for docID, freq := range posting.Docs {
			docs[docID] = Match{Token: token, Frequency: freq}
		}
		docLists = append(docLists, docs)
	}

	return docLists

}

func processANDQuery(docLists []map[string]Match) ([]Result) {
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
	return results

}

func processORQuery(docLists []map[string]Match) ([]Result) {
	result_map := make(map[string]Result, 1024)

	for _, docList := range docLists {
		for docID, match := range docList {
			result, ok := result_map[docID]
			if !ok {
				result = Result{DocID: docID}
			}
			result.TotalTermFrequency += match.Frequency
			result.Matches = append(result.Matches, match)

			result_map[docID] = result
		}
	}

	results := make([]Result, 0, len(result_map))
	for _, r := range result_map {
		results = append(results, r)
	}

	return results
}

// Search finds documents that contain all query tokens and orders them by frequency.
func Search(idx index.InvertedIndex, tokenizer text.Tokenizer, query string) ([]Result, int, error) {
	if query == "" {
		return nil, 0, fmt.Errorf("query must not be empty")
	}

	docLists := PreprocessQuery(idx, tokenizer, query)
	if len(docLists) == 0 {
		return nil, 0, nil
	}
	results := make([]Result, 0)

	if strings.Contains((query), "or") {
		// TODO: properly split at every or and support a combination of OR and AND queries
		results = processORQuery(docLists)
	} else {
		results = processANDQuery(docLists)
	}

	// sort results by frequency
	sort.Slice(results, func(i, j int) bool {
		if results[i].TotalTermFrequency == results[j].TotalTermFrequency {
			return results[i].DocID < results[j].DocID
		}
		return results[i].TotalTermFrequency > results[j].TotalTermFrequency
	})

	resultCount := len(results)

	// return top 10 results
	if len(results) > 10 {
		results = results[:10]
	}

	return results, resultCount, nil
}
