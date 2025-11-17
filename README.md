![ChatGPT Image Oct 27, 2025 at 10_54_42 AM](https://github.com/user-attachments/assets/13e8d57f-7529-4178-8d87-b95b1179d882)

# luen-search-engine

A search engine implementation in Go with support for large-scale document indexing.
Loads documents from the MS MARCO collection, builds an inverted index, 
and provides a CLI search interface across query terms.

## Prerequisites

- Go 1.25+
- The `msmarco-docs.tsv` [dataset](https://microsoft.github.io/msmarco/Datasets.html#datasets). Place it anywhere convenient – by default the program looks for `data/msmarco-docs.tsv`.

For Benchmarking:
- A benchmark subset at `data/msmarco-docs-bench-100000.tsv` (first 100k rows). You can create it with `head -n 100000 data/msmarco-docs.tsv > data/msmarco-docs-bench-100000.tsv`.
- A deterministic query list at `data/msmarco-queries-bench-1000.txt`, which is added to the repo for convenience and for reproducibility (generated your own via `scripts/generate-bench-queries.sh` requires the benchmark subset).

## Getting Started

A `makefile` is provided for common tasks:

```bash
make help          # Show available commands
make build         # Compile the binary
make run           # Build and run with default settings
make test          # Run all tests
make bench         # Run performance benchmarks
make lint          # Run linter
```

### Dataset setup

Run `scripts/preprocess_dataset.sh` before indexing to strip the leading `D` from MS MARCO document IDs and write a `-preprocessed` TSV. This makes it cheaper to store IDs as integers. It might take some time (10 minutes on Apple M1 Pro) on the full corpus.
You need to make the script executable first: `chmod +x scripts/preprocess_dataset.sh`

### Running with custom parameters

```bash
# Build and run
make build
./bin/luen -limit 100000

# Or use go run directly
go run . -limit 100000
```

### Command-line flags

- `-data` – Path to the TSV file (default: `data/msmarco-docs-preprocess.tsv`)
- `-limit` – Maximum number of documents to load (default: 1000, set to 0 for all)
- `-cpuprofile` – Path to write a CPU profile (disabled by default)
- `-memprofile` – Path to write a heap profile (disabled by default)

### Search interface

Once the index is ready, you can enter search terms at the prompt. 
Type `>exit` or press `Ctrl+D` to quit.

Example:
```
Search: machine learning algorithms
📊 Found 42 result(s)
...
```

## Profiling

- `make bench` exercises the deterministic index + query workloads described in `profile.md`.
- For detailed instructions on CLI flags, `go tool pprof`, tracing, and Perfetto-style visualization, see [profile.md](profile.md).

## Performance

See open issues for planned optimizations (parallelization, tokenization improvements).

## Project Layout

- `internal/data` – Dataset loader
- `internal/text` – Tokenizer with stopword filtering
- `internal/index` – Inverted index builder
- `internal/search` – Query evaluation and ranking
- `internal/output` – CLI rendering helpers
