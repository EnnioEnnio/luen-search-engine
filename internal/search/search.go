package search

import (
	"fmt"
	"sort"

	"luen-search-engine/internal/index"
	"luen-search-engine/internal/model"
	"luen-search-engine/internal/processing"
	"luen-search-engine/internal/text"
)

type DocID = model.DocID
type Token = model.Token
type Match = model.Match
type Result = model.Result

// Search finds documents that contain all query tokens and orders them by frequency.
func Search(src index.PostingSource, tokenizer *text.Tokenizer, query string) (res []Result, count int, e error) {
	if query == "" {
		return nil, 0, fmt.Errorf("query must not be empty")
	}

	ast, err := processing.Parse(query, tokenizer)
	if err != nil {
		return nil, 0, err
	}

	results, err := ast.Eval(src)
	if err != nil {
		return nil, 0, err
	}

	// sort results by frequency
	sort.Slice(results, func(i, j int) bool {
		if results[i].TotalTermFrequency == results[j].TotalTermFrequency {
			return results[i].DocID < results[j].DocID
		}
		return results[i].TotalTermFrequency > results[j].TotalTermFrequency
	})

	resCount := len(results)

	// return top 10 results
	if len(results) > 10 {
		results = results[:10]
	}

	return results, resCount, nil
}
