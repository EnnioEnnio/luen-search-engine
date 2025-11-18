package indexer

import (
	"testing"
	"time"

	"luen-search-engine/internal/benchutil"
	"luen-search-engine/internal/data"
	"luen-search-engine/internal/index/disk"
	"luen-search-engine/internal/search"
	"luen-search-engine/internal/text"
)

const benchBatchBytes = 64 * 1024 * 1024

func BenchmarkDiskIndexBuild(b *testing.B) {
	tokenizer := text.NewTokenizer()
	for i := 0; i < b.N; i++ {
		tempDir := b.TempDir()
		cfg := Config{
			DataPath:   benchutil.DatasetPath,
			Limit:      benchutil.DatasetLimit,
			BatchBytes: benchBatchBytes,
			OutputDir:  tempDir,
			Tokenizer:  tokenizer,
		}
		builder := NewBuilder(cfg)
		start := time.Now()
		manifest, err := builder.Build()
		if err != nil {
			b.Fatalf("build disk index: %v", err)
		}
		dur := time.Since(start)
		b.ReportMetric(float64(manifest.DocumentCount)/dur.Seconds(), "docs/s")
	}
}

func BenchmarkDiskQueryServing(b *testing.B) {
	tokenizer := text.NewTokenizer()
	dir := buildBenchIndex(b, tokenizer)
	dict, err := disk.LoadDictionary(DictionaryPath(dir))
	if err != nil {
		b.Fatalf("load dictionary: %v", err)
	}
	store, err := disk.NewPostingStore(dict, PostingsPath(dir), 4096)
	if err != nil {
		b.Fatalf("open posting store: %v", err)
	}
	b.Cleanup(func() { _ = store.Close() })
	docs, err := data.OpenDocumentStore(dir)
	if err != nil {
		b.Fatalf("open document store: %v", err)
	}
	b.Cleanup(func() { _ = docs.Close() })
	queries := benchutil.LoadQueries(b)
	b.ReportMetric(float64(dict.Size()), "tokens_loaded")
	b.Logf("dictionary entries: %d", dict.Size())
	b.ReportMetric(float64(len(queries)), "queries_per_loop")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, query := range queries {
			results, _, err := search.Search(store, tokenizer, query)
			if err != nil {
				b.Fatalf("search query %q: %v", query, err)
			}
			for _, result := range results {
				if _, ok := docs.Lookup(result.DocID); !ok {
					b.Fatalf("doc %d missing from docstore", result.DocID)
				}
			}
		}
	}
}

func buildBenchIndex(tb testing.TB, tokenizer *text.Tokenizer) string {
	tb.Helper()
	output := tb.TempDir()
	cfg := Config{
		DataPath:   benchutil.DatasetPath,
		Limit:      benchutil.DatasetLimit,
		BatchBytes: benchBatchBytes,
		OutputDir:  output,
		Tokenizer:  tokenizer,
	}
	builder := NewBuilder(cfg)
	start := time.Now()
	manifest, err := builder.Build()
	if err != nil {
		tb.Fatalf("build bench index: %v", err)
	}
	tb.Logf("disk index built in %s (%d docs, %d tokens, %d partials)", time.Since(start).Round(time.Millisecond), manifest.DocumentCount, manifest.TokenCount, manifest.PartialFiles)
	return output
}
