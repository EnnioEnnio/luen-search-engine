package search

import (
	"testing"

	"luen-search-engine/internal/benchutil"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/text"
)

func BenchmarkSearchWorkload(b *testing.B) {
	dataset := benchutil.LoadDataset(b)
	tokenizer := text.NewTokenizer()
	idx := index.Build(dataset.Documents, tokenizer)
	queries := benchutil.LoadQueries(b)
	if len(queries) == 0 {
		b.Fatal("benchmark queries must not be empty")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, query := range queries {
			if _, _, err := Search(idx, tokenizer, query, "single"); err != nil {
				b.Fatalf("search query %q returned error: %v", query, err)
			}
		}
	}

	b.ReportMetric(float64(len(queries)), "queries")
}
