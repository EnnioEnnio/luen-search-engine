package indexer

import (
	"os"
	"sync"
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
	datasetPath := benchutil.DatasetPathAbs(b)
	var lastManifest *Manifest
	var lastDuration time.Duration
	for i := 0; i < b.N; i++ {
		outputDir := b.TempDir()
		markShared := false
		if i == 0 {
			sharedIndexMu.Lock()
			if sharedIndexPath == "" {
				dir, err := os.MkdirTemp("", "disk-bench-shared-")
				if err != nil {
					sharedIndexMu.Unlock()
					b.Fatalf("create shared temp dir: %v", err)
				}
				outputDir = dir
				markShared = true
			}
			sharedIndexMu.Unlock()
		}
		manifest, dur := runSingleBuildBenchmark(b, outputDir, tokenizer, datasetPath)
		lastManifest = manifest
		lastDuration = dur
		if markShared {
			setSharedIndexInfo(outputDir, manifest, dur)
		}
	}
	if lastManifest != nil {
		fields := map[string]interface{}{
			"documents":     lastManifest.DocumentCount,
			"unique_tokens": lastManifest.TokenCount,
			"partials":      lastManifest.PartialFiles,
			"build_time":    lastDuration,
			"iterations":    b.N,
		}
		if lastDuration > 0 {
			fields["docs_per_sec"] = float64(lastManifest.DocumentCount) / lastDuration.Seconds()
		}
		benchutil.LogBenchmarkSummary(b, "DiskIndexBuild", fields)
	}
}

func BenchmarkDiskQueryServing(b *testing.B) {
	tokenizer := text.NewTokenizer()
	datasetPath := benchutil.DatasetPathAbs(b)
	dir, manifest := ensureSharedBenchIndex(b, tokenizer, datasetPath)
	registerSharedCleanup(b, dir)
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
	b.ReportMetric(float64(len(queries)), "queries_per_loop")

	loopStart := time.Now()
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
	b.StopTimer()
	loopDuration := time.Since(loopStart)
	queriesPerLoop := len(queries)
	totalQueries := queriesPerLoop * b.N
	avgPerQuery := time.Duration(0)
	if totalQueries > 0 {
		avgPerQuery = loopDuration / time.Duration(totalQueries)
	}
	fields := map[string]interface{}{
		"queries_per_loop": queriesPerLoop,
		"tokens_loaded":    dict.Size(),
		"total_queries":    totalQueries,
		"loop_time":        loopDuration,
		"avg_per_query":    avgPerQuery,
	}
	if manifest != nil {
		fields["documents_indexed"] = manifest.DocumentCount
		fields["index_build_time"] = sharedIndexBuildTime
	}
	benchutil.LogBenchmarkSummary(b, "DiskQueryServing", fields)
}

var (
	sharedIndexMu        sync.Mutex
	sharedIndexPath      string
	sharedIndexManifest  *Manifest
	sharedIndexBuildTime time.Duration
	sharedCleanupOnce    sync.Once
)

func setSharedIndexInfo(path string, manifest *Manifest, dur time.Duration) {
	sharedIndexMu.Lock()
	if sharedIndexPath == "" {
		sharedIndexPath = path
		sharedIndexManifest = manifest
		sharedIndexBuildTime = dur
	}
	sharedIndexMu.Unlock()
}

func ensureSharedBenchIndex(tb testing.TB, tokenizer *text.Tokenizer, datasetPath string) (string, *Manifest) {
	sharedIndexMu.Lock()
	path := sharedIndexPath
	manifest := sharedIndexManifest
	sharedIndexMu.Unlock()
	if path != "" && manifest != nil {
		return path, manifest
	}
	dir, err := os.MkdirTemp("", "disk-bench-shared-")
	if err != nil {
		tb.Fatalf("create shared temp dir: %v", err)
	}
	manifest, dur := runSingleBuildBenchmark(tb, dir, tokenizer, datasetPath)
	setSharedIndexInfo(dir, manifest, dur)
	return dir, manifest
}

func registerSharedCleanup(tb testing.TB, path string) {
	if path == "" {
		return
	}
	sharedCleanupOnce.Do(func() {
		tb.Cleanup(func() {
			_ = os.RemoveAll(path)
		})
	})
}

func runSingleBuildBenchmark(tb testing.TB, output string, tokenizer *text.Tokenizer, datasetPath string) (*Manifest, time.Duration) {
	builder := NewBuilder(Config{
		DataPath:   datasetPath,
		Limit:      benchutil.DatasetLimit,
		BatchBytes: benchBatchBytes,
		OutputDir:  output,
		Tokenizer:  tokenizer,
	})
	start := time.Now()
	var (
		manifest *Manifest
		err      error
	)
	benchutil.WithMutedLogs(tb, func() {
		manifest, err = builder.Build()
	})
	if err != nil {
		tb.Fatalf("build bench index: %v", err)
	}
	dur := time.Since(start)
	if b, ok := tb.(*testing.B); ok {
		if dur > 0 {
			b.ReportMetric(float64(manifest.DocumentCount)/dur.Seconds(), "docs/s")
		}
	}
	return manifest, dur
}
