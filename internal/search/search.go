package search

import (
	"context"
	"fmt"
	"sort"

	"luen-search-engine/internal/index/disk"
	"luen-search-engine/internal/model"
	"luen-search-engine/internal/processing"
	"luen-search-engine/internal/synonyms"
	"luen-search-engine/internal/text"
)

type DocID = model.DocID
type Token = model.Token
type Match = model.Match
type Result = model.Result

// Search finds documents that contain all query tokens and orders them by frequency.
func Search(ctx context.Context, src *disk.PostingStore, tokenizer *text.Tokenizer, query string, expander ...*synonyms.SpladeLike) (res []Result, count int, e error) {
	if query == "" {
		return nil, 0, fmt.Errorf("query must not be empty")
	}

	ast, err := processing.Parse(query, tokenizer)
	if err != nil {
		return nil, 0, err
	}
	// an expander is only provided when synonym expansion is enabled explicitly with a flag
	if expander != nil {
		ast, err = expander[0].ExpandAST(ctx, ast, query)
		if err != nil {
			return nil, 0, err
		}
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
