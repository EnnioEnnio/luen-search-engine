package search

import (
	"context"
	"sync"
	"testing"
	"time"

	"luen-search-engine/internal/benchutil"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/processing"
	"luen-search-engine/internal/text"
)

var (
	inMemoryQueryOnce  sync.Once
	inMemorySource     *processing.MemorySource
	inMemoryDocs       int
	inMemoryTokens     int
	inMemoryDocLengths map[uint32]uint32
)

func BenchmarkSearchWorkload(b *testing.B) {
	dataset := benchutil.LoadDataset(b)
	tokenizer := text.NewTokenizer()
	inMemoryQueryOnce.Do(func() {
		buildResult := index.Build(dataset.Documents, tokenizer)
		inMemorySource = processing.NewMemorySource(buildResult.Index)
		inMemoryDocs = len(dataset.Documents)
		inMemoryTokens = buildResult.Index.TokenCount()
		inMemoryDocLengths = buildResult.DocLengths
	})
	if inMemorySource == nil {
		b.Fatal("failed to prepare in-memory posting source")
	}
	queries := benchutil.LoadQueries(b)
	if len(queries) == 0 {
		b.Fatal("benchmark queries must not be empty")
	}

	b.ReportAllocs()
	loopStart := time.Now()
	b.ResetTimer()

	source := inMemorySource
	for i := 0; i < b.N; i++ {
		for _, query := range queries {
			if _, _, err := Search(context.Background(), source, tokenizer, query, inMemoryDocLengths); err != nil {
				b.Fatalf("search query %q returned error: %v", query, err)
			}
		}
	}
	b.StopTimer()
	loopDuration := time.Since(loopStart)

	b.ReportMetric(float64(len(queries)), "queries")
	totalQueries := len(queries) * b.N
	avgPerQuery := time.Duration(0)
	if totalQueries > 0 {
		avgPerQuery = loopDuration / time.Duration(totalQueries)
	}
	benchutil.LogBenchmarkSummary(b, "InMemoryQueryServing", map[string]interface{}{
		"documents":        inMemoryDocs,
		"unique_tokens":    inMemoryTokens,
		"queries_per_loop": len(queries),
		"total_queries":    totalQueries,
		"loop_time":        loopDuration,
		"avg_per_query":    avgPerQuery,
	})
}
