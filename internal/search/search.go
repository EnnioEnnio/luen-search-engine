package search

import (
	"context"
	"fmt"
	"sort"

	"luen-search-engine/internal/index"
	"luen-search-engine/internal/index/disk"
	"luen-search-engine/internal/model"
	"luen-search-engine/internal/pb"
	"luen-search-engine/internal/processing"
	"luen-search-engine/internal/ranking"
	"luen-search-engine/internal/synonyms"
	"luen-search-engine/internal/text"
)

type DocID = model.DocID
type Token = model.Token
type Match = model.Match
type Result = model.Result
type FieldDocLength = model.FieldDocLengths

type Response struct {
	Results []Result
	Count   int
	Error   error
}

// Search finds documents that contain all query tokens and orders them by frequency.
func Search(ctx context.Context, src *disk.PostingStore, tokenizer *text.Tokenizer, semanticClient SemanticSearcher, query string, docLengths map[DocID]FieldDocLength, expander ...*synonyms.SpladeLike) (semanticResults []Result, bm25Results []Result, bm25Count int, e error) {
	if query == "" {
		return nil, nil, 0, fmt.Errorf("query must not be empty")
	}
	BM25Results := make(chan Response, 1)
	SemanticResults := make(chan Response, 1)

	// BM25 search
	go func() {
		results, resCount, err := SearchWithBM25(ctx, src, tokenizer, query, docLengths, expander...)
		BM25Results <- Response{Results: results, Count: resCount, Error: err}
	}()
	// Semantic search
	go func() {
		results, resCount, err := SemanticSearch(ctx, semanticClient, query)
		SemanticResults <- Response{Results: results, Count: resCount, Error: err}
	}()

	bm25Res := <-BM25Results
	if bm25Res.Error != nil {
		return nil, nil, 0, bm25Res.Error
	}
	semanticRes := <-SemanticResults
	if semanticRes.Error != nil {
		return nil, nil, 0, semanticRes.Error
	}

	// return top 10 bm25-results
	if len(bm25Res.Results) > 10 {
		bm25Res.Results = bm25Res.Results[:10]
	}

	return semanticRes.Results, bm25Res.Results, bm25Res.Count, nil
}

func SearchWithBM25(ctx context.Context, src *disk.PostingStore, tokenizer *text.Tokenizer, query string, docLengths map[DocID]FieldDocLength, expander ...*synonyms.SpladeLike) (res []Result, count int, e error) {

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
		if err != nil {
			return nil, 0, fmt.Errorf("get posting list for token %q: %w", token, err)
		}
		if posting != nil {
			DocumentFrequencies[token] = len(posting.Docs)
		} else {
			DocumentFrequencies[token] = 0
		}
	}

	documentCount := len(docLengths)

	// IDF-threshold optimization: only prune low-IDF terms for larger document sets
	if documentCount >= 1000 {
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
	}

	results, err := ast.Eval(src)
	if err != nil {
		return nil, 0, err
	}

	// Split positions into title and body for each match
	for i := range results {
		for j := range results[i].Matches {
			match := &results[i].Matches[j]
			fieldLengths := docLengths[results[i].DocID]
			titleBoundary := int(fieldLengths.TitleLength)

			for _, pos := range match.Positions {
				if pos < titleBoundary {
					match.TF_title++
				} else {
					match.TF_body++
				}
			}
		}
	}

	avgTitleLength, avgBodyLength := index.CalculateAvgFieldLengths(docLengths)
	documentCount = len(docLengths)

	// calculate BM25 scores
	for i, result := range results {
		bm25Score := 0.0
		for _, match := range result.Matches {
			fieldLengths := docLengths[result.DocID]
			bm25Score += ranking.CalculateFieldedBM25Score(
				match.TF_title, match.TF_body,
				fieldLengths.TitleLength, fieldLengths.BodyLength,
				avgTitleLength, avgBodyLength,
				DocumentFrequencies[match.Token], documentCount)
		}
		results[i].Score = bm25Score
	}
	// sort results by bm25 score
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].DocID < results[j].DocID
		}
		return results[i].Score > results[j].Score
	})
	resCount := len(results)

	return results, resCount, nil
}

// SemanticSearcher defines the interface for semantic search operations.
// This interface allows for easy mocking in tests.
type SemanticSearcher interface {
	Search(ctx context.Context, query string, k int32) ([]Result, error)
}

// Ensure the real implementation satisfies the interface
// Note: The actual gRPC client (pb.SemanticEmbeddingServiceClient) doesn't perfectly match this signature
// so we'll likely need a wrapper struct if we want to strictly use this interface,
// or we can pass the pb client directly if we don't mind coupling.
// Better approach: Define the interface we want to use in our core logic.

// SemanticClient wraps the gRPC client to satisfy the SemanticSearcher interface.
type SemanticClient struct {
	Client pb.SemanticEmbeddingServiceClient
}

func (s *SemanticClient) Search(ctx context.Context, query string, k int32) ([]Result, error) {
	resp, err := s.Client.Search(ctx, &pb.SearchRequest{
		Query: query,
		K:     k,
	})
	if err != nil {
		return nil, err
	}

	results := make([]Result, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = Result{
			DocID: DocID(r.DocId),
			Score: float64(r.Score),
		}
	}
	return results, nil
}

// SemanticSearch performs a semantic search using the provided client.
func SemanticSearch(ctx context.Context, client SemanticSearcher, query string) (results []Result, count int, e error) {
	if client == nil {
		// Log warning or return empty if no client configured
		return nil, 0, nil
	}

	// Request top 100 results from semantic search to allow for re-ranking or broader context
	semanticResults, err := client.Search(ctx, query, 100)
	if err != nil {
		return nil, 0, fmt.Errorf("semantic search failed: %w", err)
	}

	return semanticResults, len(semanticResults), nil
}
