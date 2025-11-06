package search

import (
	"fmt"
	"slices"
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
	TotalTermFrequency int // used for ranking only
	Matches            []Match
}

// PreprocessQuery returns a List of maps(key: docID, value: Match), for each token one list
func PreprocessQuery(idx index.InvertedIndex, tokenizer text.Tokenizer, query string) ([]map[string]Match, []bool) {
	tokens, isNegated := tokenizer.Tokenize(query)
	docLists := make([]map[string]Match, 0, len(tokens))

	if len(tokens) == 0 {
		return docLists, isNegated
	}

	// Sort Tokens to ensure non-negated token (false values) before negated token (true values)
	sort.SliceStable(tokens, func(i, j int) bool {
		// less interface is i < j -> true -> i before j
		// we want not negated (false) before negated (true), thus i = false && j = true -> true
		return !isNegated[i] && isNegated[j]
	})

	sort.SliceStable(isNegated, func(i, j int) bool {
		// see sorting above
		return !isNegated[i] && isNegated[j]
	})

	for _, token := range tokens {
		posting, found := idx[token]
		if !found {
			continue
		}

		matches := make(map[string]Match, len(posting.Docs))
		for docID, freq := range posting.Docs {
			matches[docID] = Match{Token: token, Frequency: freq}
		}
		docLists = append(docLists, matches)
	}

	return docLists, isNegated
}

func processANDQuery(docLists []map[string]Match, isNegated []bool) []Result {
	results := make([]Result, 0)

	for docID, headMatch := range docLists[0] { // iterate over first token's results
		total := headMatch.Frequency
		matches := []Match{headMatch}
		missing := false
		excludeDoc := false

		for i, docs := range docLists[1:] { // iterate over other tokens results
			actualIndex := i + 1

			if isNegated[actualIndex] {
				if _, found := docs[docID]; found {
					excludeDoc = true
					break
				}
				continue
			}

			match, found := docs[docID]
			if !found {
				missing = true
				break
			}
			total += match.Frequency
			matches = append(matches, match)
		}

		if missing || excludeDoc {
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

func processORQuery(docLists []map[string]Match) []Result {
	result_map := make(map[string]Result, 0)

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

	docLists, isNegated := PreprocessQuery(idx, tokenizer, query)
	if len(docLists) == 0 {
		return nil, 0, nil
	}

	if !slices.Contains(isNegated, false) {
		return nil, 0, fmt.Errorf("query contains only negations: at least one positive search term is required")
	}

	results := make([]Result, 0)

	if strings.Contains(query, "or") {
		if slices.Contains(isNegated, true) {
			return nil, 0, fmt.Errorf("combining NOT and OR queries is not allowed")
		}
		// TODO: properly split at every or and support a combination of OR and AND queries
		results = processORQuery(docLists)
	} else {
		results = processANDQuery(docLists, isNegated)
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
