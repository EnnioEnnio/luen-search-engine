![ChatGPT Image Oct 27, 2025 at 10_54_42 AM](https://github.com/user-attachments/assets/13e8d57f-7529-4178-8d87-b95b1179d882)

# luen-search-engine

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Getting Started](#getting-started)
3. [Dataset setup](#dataset-setup)
4. [Running with custom parameters](#running-with-custom-parameters)
5. [Command-line flags](#command-line-flags)
6. [External/blocked indexing](#externalblocked-indexing)
7. [Disk-based serving](#disk-based-serving)
8. [Searching](#searching)
9. [Profiling](#profiling)
10. [Performance](#performance)
11. [Project Layout](#project-layout)

A search engine implementation in Go with support for large-scale document indexing.
Loads documents from the MS MARCO collection, builds an inverted index, 
and provides a CLI search interface across query terms.

## Prerequisites

- Go 1.25+
- The `msmarco-docs.tsv` [dataset](https://microsoft.github.io/msmarco/Datasets.html#datasets). Place it anywhere convenient – by default the program looks for `data/msmarco-docs-preprocessed.tsv`.

For Benchmarking:
- A benchmark subset at `data/msmarco-docs-bench-100000.tsv` (first 100k rows). You can create it with `head -n 100000 data/msmarco-docs.tsv > data/msmarco-docs-bench-100000.tsv`.
- A deterministic query list at `data/msmarco-queries-bench-1000.txt`, which is added to the repo for convenience and for reproducibility (generated your own via `scripts/generate-bench-queries.sh` requires the benchmark subset).

## Getting Started

A `makefile` is provided for common tasks:

```bash
make help    # Show available commands
make build   # Compile the binary into bin/luen
make dev     # Run the CLI in in-memory mode (default limit 10k docs)
make run     # Run in disk-serving mode (auto-builds 100k docs if needed)
make index   # Force a full on-disk index build (limit=0)
make test    # Run all tests
make bench   # Run benchmark suites (100k dataset + 1k queries)
make lint    # Run go vet
make tidy    # go mod tidy
make clean   # Remove build artifacts and index directory
```

### Dataset setup

Run `scripts/preprocess_dataset.sh` before indexing to strip the leading `D` from MS MARCO document IDs and write a `-preprocessed` TSV. This makes it cheaper to store IDs as integers. It might take some time (10 minutes on Apple M1 Pro) on the full corpus.
You need to make the script executable first: `chmod +x scripts/preprocess_dataset.sh`

### Running with custom parameters

```bash
# Build and run in-memory with a custom limit
make build
./bin/luen -disk=false -limit 5000

# Force a full disk index build and then serve from disk
./bin/luen -buildindex -indexdir index -limit 0
./bin/luen -disk=true -indexdir index -disklimit 100000

# Or use go run directly
go run . -disk=true -disklimit 100000
```

### Command-line flags

- `-data` – Path to the TSV file (default: `data/msmarco-docs-preprocessed.tsv`)
- `-limit` – Maximum number of documents to load (default: 1000, set to 0 for all)
- `-buildindex` – Build the SPIMI-based on-disk index and exit (default: false)
- `-disk` – Serve queries straight from the persisted index (default: false)
- `-disklimit` – When `-disk` is enabled, auto-build this many docs if the index is missing (default: 100 k)
- `-diskcache` – Posting-list cache size (number of tokens) while serving from disk (default: 2048)
- `-indexdir` – Output directory for the on-disk index (default: `index`)
- `-indexbatch` – Approximate batch size in bytes for blocked indexing (default: 64 MB)
- `-cpuprofile` – Path to write a CPU profile (disabled by default)
- `-memprofile` – Path to write a heap profile (disabled by default)

### External/blocked indexing

Build the on-disk index in bounded batches to keep heap usage under 1 GB:

```bash
GODEBUG=gctrace=1 go run . -data data/msmarco-docs-preprocessed.tsv -buildindex -indexdir index -indexbatch $((64*1024*1024))
```

Each batch loads only a few MB of documents (default: 64 MB to not exceed 1GB of RAM), builds a partial inverted index in memory, spills it to disk, and finally performs a multi-way merge into `postings.bin`/`dictionary.tsv` under `-indexdir`. During the same pass we also write `docs.bin`/`docs.idx`, which store the raw MS MARCO rows and an offset table so we can fetch titles/bodies later without keeping them all in RAM. Monitoring `gctrace` while tuning the `-indexbatch` threshold keeps the heap within the desired budget.

### Disk-based serving

Running `make run` (or `./bin/luen -disk`) bootstraps the CLI straight from the on-disk assets:

1. If `dictionary.tsv`, `postings.bin`, `docs.bin`, or `docs.idx` are missing, the program triggers the external builder with `-disklimit` documents (default 100 k) before serving queries.
2. Startup only loads the dictionary/manifest + doc-index metadata, which keeps warm-up <10 s even for millions of tokens.
3. At query time we pull just the required posting lists via offsets inside `postings.bin`, caching the hottest ones in an LRU of configurable size (`-diskcache`).
4. After ranking, we look up the top-k document payloads by seeking into `docs.bin` using its offset index, so doc rendering doesn’t force full-dataset residency.

`make dev` continues to run the in-memory codepath for faster iteration on small corpora, while `make run` ensures the persistent index is reused across runs.

### Searching

The CLI accepts a small boolean grammar inspired by typical search engines:

- **Implicit AND**: entering `machine learning` requires both tokens (equivalent to `machine and learning`). Token order does not matter for implicit AND.
- **Explicit AND / OR**: `cat and dog or bird` matches docs containing both `cat` and `dog`, or any doc containing `bird`.
- **NOT**: `not` negates the immediately following term or group. Example: `cat and not dog` returns docs containing `cat` but not `dog`. `NOT` cannot stand alone; it must be combined with a positive operand via `and`.
- **Phrase queries**: wrap tokens in double quotes to search for contiguous sequences, e.g., `"machine learning"` requires exact token order.
- **Parentheses**: override default precedence (not has the highest precendence, implicit AND has higher precedence than OR). Example: `(cat or dog) and bird` ensures the OR is evaluated before the AND.

### Search interface

Once the index is ready, you can enter search terms at the prompt. 
Type `>exit` or press `Ctrl+D` to quit.

Example:
```
Search: "New York"

📊 Found 9197 result(s) in 42.236625ms
════════════════════════════════════════════════════════════════════════════════

[1] Office of the Professions
    🔗 http://www.op.nysed.gov/prof/sw/swceproviderlist.htm
    📝 Matches:
       new count: 192
       york count: 192
────────────────────────────────────────────────────────────────────────────────

[2] ...
```

## Profiling

- `make bench` exercises both the in-memory search benchmarks and the new disk pipeline benchmarks (build + serve against the 100k dataset/1k queries). The disk suite reports docs/sec for the builder along with dictionary size metrics and verifies doc fetches from `docs.bin`.
- For detailed instructions on CLI flags, `go tool pprof`, tracing, and Perfetto-style visualization, see [profile.md](profile.md).

## Performance

See open issues for planned optimizations (parallelization, tokenization improvements).

## Project Layout

- `internal/data` – Dataset loader
- `internal/text` – Tokenizer with stopword filtering
- `internal/index` – Inverted index builder
- `internal/search` – Query evaluation and ranking
- `internal/output` – CLI rendering helpers
