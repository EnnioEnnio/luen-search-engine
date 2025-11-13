package index

import (
	"testing"

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
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		idx := Build(docs, tokenizer)
		if idx.TokenCount() == 0 {
			b.Fatal("unexpected empty index")
		}
	}

	b.ReportMetric(float64(len(docs)), "documents")
}
