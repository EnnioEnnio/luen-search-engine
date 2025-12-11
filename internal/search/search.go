package search

import (
	"context"
	"fmt"
	"sort"

	"luen-search-engine/internal/index"
	"luen-search-engine/internal/index/disk"
	"luen-search-engine/internal/model"
	"luen-search-engine/internal/processing"
	"luen-search-engine/internal/ranking"
	"luen-search-engine/internal/synonyms"
	"luen-search-engine/internal/text"
)

type DocID = model.DocID
type Token = model.Token
type Match = model.Match
type Result = model.Result
type DocLength = model.DocLength

// Search finds documents that contain all query tokens and orders them by frequency.
func Search(ctx context.Context, src *disk.PostingStore, tokenizer *text.Tokenizer, query string, docLengths map[DocID]DocLength, expander ...*synonyms.SpladeLike) (res []Result, count int, e error) {
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
	// prepare BM25 calculation
	DocumentFrequencies := make(map[Token]int)
	tokens := ast.GetPositiveTokens()
	if len(tokens) == 0 {
		return nil, 0, fmt.Errorf("query must contain at least one positive term")
	}
	for _, token := range tokens {
		// TODO: it would be more efficient to only call a function that returns the document frequency instead of the full posting list
		posting, err := src.Lookup(token)
		if err != nil || posting == nil {
			return nil, 0, fmt.Errorf("get posting list for token %q: %w", token, err)
		}
		DocumentFrequencies[token] = len(posting.Docs)
	}

	avgDocLength := index.CalculateAvgDocLength(docLengths)
	documentCount := len(docLengths)

	highIDFTokens := make(map[Token]bool)
	threshold := 1.5
	for _, token := range tokens {
		idf := ranking.CalculateIDF(documentCount, DocumentFrequencies[token])
		if idf > threshold {
			highIDFTokens[token] = true
		}
	}
	ast = ast.Prune(highIDFTokens)

	if ast == nil {
		return nil, 0, nil
	}

	results, err := ast.Eval(src)
	if err != nil {
		return nil, 0, err
	}

	// calculate BM25 scores
	for i, result := range results {
		bm25Score := 0.0
		for _, match := range result.Matches {

			bm25Score += ranking.CalculateBM25Score(match.Frequency, docLengths[result.DocID], avgDocLength, DocumentFrequencies[match.Token], documentCount)
		}
		results[i].BM25Score = bm25Score
	}
	// sort results by bm25 score
	sort.Slice(results, func(i, j int) bool {
		if results[i].BM25Score == results[j].BM25Score {
			return results[i].DocID < results[j].DocID
		}
		return results[i].BM25Score > results[j].BM25Score
	})

	resCount := len(results)

	// return top 10 results
	if len(results) > 10 {
		results = results[:10]
	}

	return results, resCount, nil
}
