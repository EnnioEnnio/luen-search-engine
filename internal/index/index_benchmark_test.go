package index

import (
	"testing"
	"time"

	"luen-search-engine/internal/benchutil"
	"luen-search-engine/internal/text"
)

func BenchmarkBuildMSMarco100k(b *testing.B) {
	tokenizer := text.NewTokenizer()
	dataset := benchutil.LoadDataset(b)
	docs := dataset.Documents
	if len(docs) == 0 {
		b.Fatal("benchmark dataset returned no documents")
	}

	b.ReportAllocs()
	start := time.Now()
	b.ResetTimer()

	var lastTokenCount int
	for i := 0; i < b.N; i++ {
		buildResult := Build(docs, tokenizer)
		if buildResult.Index.TokenCount() == 0 {
			b.Fatal("unexpected empty index")
		}
		lastTokenCount = buildResult.Index.TokenCount()
	}
	b.StopTimer()

	b.ReportMetric(float64(len(docs)), "documents")
	dur := time.Since(start)
	benchutil.LogBenchmarkSummary(b, "InMemoryIndexBuild", map[string]interface{}{
		"documents":     len(docs),
		"unique_tokens": lastTokenCount,
		"iterations":    b.N,
		"total_time":    dur,
		"avg_per_iter":  dur / time.Duration(b.N),
	})
}
