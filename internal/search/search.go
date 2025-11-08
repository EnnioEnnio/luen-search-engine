package search

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"luen-search-engine/internal/index"
	"luen-search-engine/internal/model"
	"luen-search-engine/internal/text"
)

type DocID = model.DocID
type Token = model.Token

// Match keeps track of how often a token occurred in a document.
type Match struct {
	Token     Token
	Frequency int
	Positions []int
}

// Result represents the aggregated scores for a matching document.
type Result struct {
	DocID              DocID
	TotalTermFrequency int // used for ranking only
	Matches            []Match
}

// PreprocessQuery returns a list of maps(key: docID, value: Match), for each token one list
func PreprocessQuery(idx index.InvertedIndex, tokenizer text.Tokenizer, query string) ([]map[DocID]Match, []bool) {
	tokens, isNegated := tokenizer.Tokenize(query)
	docLists := make([]map[DocID]Match, 0, len(tokens))

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

		matches := make(map[DocID]Match, len(posting.Docs))
		for docID, positions := range posting.Docs {
			matches[docID] = Match{Token: token, Frequency: len(positions), Positions: positions}
		}
		docLists = append(docLists, matches)
	}

	return docLists, isNegated
}

func processANDQuery(docLists []map[DocID]Match, isNegated []bool) []Result {
	var results []Result
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

func processORQuery(docLists []map[DocID]Match) []Result {
	result_map := make(map[DocID]Result, 0)

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

func processPhraseQuery(docLists []map[DocID]Match) []Result {
	results := make([]Result, 0)

	// For each document that contains the first token, try to find phrase occurrences
	for docID := range docLists[0] {
		// collect the position lists for each token in the phrase for this doc
		tokenCount := len(docLists)
		positions := make([][]int, tokenCount)
		tokens := make([]Token, tokenCount)
		missing := false

		for ti := 0; ti < tokenCount; ti++ {
			match, ok := docLists[ti][docID]
			if !ok {
				missing = true
				break
			}
			positions[ti] = match.Positions
			tokens[ti] = match.Token
		}
		if missing {
			continue
		}

		// prepare output matches (one per token) with empty positions to fill
		outMatches := make([]Match, tokenCount)
		for ti := 0; ti < tokenCount; ti++ {
			outMatches[ti] = Match{Token: tokens[ti], Positions: make([]int, 0)}
		}

		// pointers into each positions list
		idxs := make([]int, tokenCount)

	OUTER:
		for {
			// if any pointer is out of bounds, we're done
			for ti := 0; ti < tokenCount; ti++ {
				if idxs[ti] >= len(positions[ti]) {
					break OUTER
				}
			}

			// current positions
			currentPositions := make([]int, tokenCount)
			for ti := 0; ti < tokenCount; ti++ {
				currentPositions[ti] = positions[ti][idxs[ti]]
			}

			// check whether the current positions form a phrase
			ok := true
			for ti := 1; ti < tokenCount; ti++ {
				if currentPositions[ti] != currentPositions[0]+ti {
					ok = false
					break
				}
			}

			if ok {
				// record the matching positions for each token
				for ti := 0; ti < tokenCount; ti++ {
					outMatches[ti].Positions = append(outMatches[ti].Positions, currentPositions[ti])
				}
				// advance all pointers to look for next (this allows overlapping phrases)
				for ti := 0; ti < tokenCount; ti++ {
					idxs[ti]++
				}
				continue
			}

			// not a match: advance the pointer(s) at the minimum current position to try to align
			minPos := currentPositions[0]
			minIdx := 0
			for ti := 1; ti < tokenCount; ti++ {
				if currentPositions[ti] < minPos {
					minPos = currentPositions[ti]
					minIdx = ti
				}
			}
			idxs[minIdx]++
		}

		if len(outMatches[0].Positions) > 0 {
			// set frequencies and total term frequency (number of phrase occurrences)
			total := len(outMatches[0].Positions)
			for ti := 0; ti < tokenCount; ti++ {
				outMatches[ti].Frequency = len(outMatches[ti].Positions)
			}

			results = append(results, Result{
				DocID:              docID,
				TotalTermFrequency: total,
				Matches:            outMatches,
			})
		}
	}

	return results

}

// Search finds documents that contain all query tokens and orders them by frequency.
func Search(idx index.InvertedIndex, tokenizer text.Tokenizer, query string, mode string) ([]Result, int, error) {
	if query == "" {
		return nil, 0, fmt.Errorf("query must not be empty")
	}

	if mode == "phrase" {
		return SearchPhrase(idx, tokenizer, query)
	}

	// Check for OR operator before tokenization (tokenizer removes stopwords)
	isORQuery := false
	lowerQuery := strings.ToLower(query)
	// Split by whitespace and check if any token is exactly "or"
	queryTokens := strings.Fields(lowerQuery)
	for _, token := range queryTokens {
		if token == "or" || token == "-or" {
			isORQuery = true
			break
		}
	}

	docLists, isNegated := PreprocessQuery(idx, tokenizer, query)
	if len(docLists) == 0 {
		return nil, 0, nil
	}

	if !slices.Contains(isNegated, false) {
		return nil, 0, fmt.Errorf("query contains only negations: at least one positive search term is required")
	}

	results := make([]Result, 0)

	if isORQuery {
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

// TO BE REMOVED!!!!!
// This is just a temporary workaround until the query parser is implemented to properly handle phrase queries within the full query string.
func SearchPhrase(idx index.InvertedIndex, tokenizer text.Tokenizer, query string) ([]Result, int, error) {
	lowerQuery := strings.ToLower(query)
	// Split by whitespace and check if any token is exactly "or"
	queryTokens := strings.Fields(lowerQuery)
	for _, token := range queryTokens {
		if token == "or" || token == "-or" {
			return nil, 0, fmt.Errorf("OR operator is not supported in phrase queries")
		}
	}

	docLists, isNegated := PreprocessQuery(idx, tokenizer, query)
	if len(docLists) == 0 {
		return nil, 0, nil
	}

	if slices.Contains(isNegated, true) {
		return nil, 0, fmt.Errorf("phrase queries do not support negation")
	}

	results := processPhraseQuery(docLists)

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
