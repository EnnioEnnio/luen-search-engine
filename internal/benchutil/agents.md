# Benchmarking Guide for Luen Search

This file explains how the automated benchmarks are structured so future agents can extend them without breaking parity between in‑memory and on‑disk workloads.

## Dataset & Query Inputs

- `internal/benchutil` exposes helpers (`LoadDataset`, `LoadQueries`) that load the 100k MS MARCO subset and 1k deterministic queries from `data/`.
- Paths are resolved relative to the repo root; callers never hard-code absolute paths.
- The dataset helper caches the parsed `*data.Dataset` so repeated benchmarks do not reread the TSV.

## Benchmark Pairs

We treat indexing and querying as pairs:

1. **In-Memory Build** – `internal/index/index_benchmark_test.go` (package `index`).
   - Builds an inverted index from the 100k dataset in each benchmark iteration.
   - Reports documents, unique tokens, iterations, and average build time.
2. **In-Memory Query** – `internal/search/search_benchmark_test.go` (package `search`).
   - Uses a `sync.Once` to build the in-memory index once via `index.Build` and wraps it in `processing.NewMemorySource` for the entire benchmark run. This mirrors the disk benchmarks, which reuse their persisted artifacts.
   - Replays the 1k queries against that cached posting source and reports loop time + average latency.

3. **Disk Build** – `internal/indexer/disk_benchmark_test.go` (`BenchmarkDiskIndexBuild`).
   - Runs the SPIMI/blocking builder with 64 MB batches, writing `postings.bin`, `dictionary.tsv`, `docs.bin`, and `docs.idx` into a temp dir.
   - The first iteration also records the output path, manifest, and build time so the query benchmark can reuse it.
4. **Disk Query** – same file (`BenchmarkDiskQueryServing`).
   - Calls `ensureSharedBenchIndex` to obtain the cached directory (building once if the build bench did not run).
   - Loads the dictionary/posting store/doc store, runs the 1k queries, and cleans up the shared temp directory after the benchmark completes.

Each benchmark logs a normalized summary block via `benchutil.LogBenchmarkSummary` so `go test -bench` output stays consistent.

## Adding New Benchmarks

- Keep the build/query pairing concept: if a query benchmark benefits from cached state, ensure the corresponding build benchmark (or the query benchmark itself) initializes it via `sync.Once` so both run under similar assumptions.
- Use `benchutil.WithMutedLogs` when running long-running builders to avoid spamming batch progress in benchmark output.
- Report key counts/metrics through `LogBenchmarkSummary` for readability.
- Never overwrite the persisted index in `index/`; use temp directories (`b.TempDir()` or `os.MkdirTemp`) and clean up via `tb.Cleanup`.

Following this structure keeps in-memory and on-disk benchmarks comparable while preventing accidental destruction of user data.
