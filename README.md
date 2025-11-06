![ChatGPT Image Oct 27, 2025 at 10_54_42 AM](https://github.com/user-attachments/assets/13e8d57f-7529-4178-8d87-b95b1179d882)

# luen-search-engine

A memory-efficient search engine implementation in Go with support for large-scale document indexing.
Loads documents from the MS MARCO collection in configurable batches, builds an inverted index, 
and provides a CLI search interface with AND semantics across query terms.

## Features

- **Batch Processing**: Process millions of documents without loading them all into memory
- **Memory Efficient**: Stores only document metadata (ID, URL, Title) after indexing
- **Configurable**: Adjustable batch sizes and document limits
- **Fast Search**: Inverted index with TF-IDF ranking

## Prerequisites

- Go 1.22+
- The `msmarco-docs.tsv` [dataset](https://microsoft.github.io/msmarco/Datasets.html#datasets). Place it anywhere convenient – by default the program looks for `data/msmarco-docs.tsv`.

## Getting Started

A `makefile` is provided for common tasks:

```bash
make help          # Show available commands
make build         # Compile the binary
make run           # Build and run with default settings
make test          # Run all tests
make lint          # Run linter
```

### Running with custom parameters

```bash
# Build and run
make build
./bin/luen -limit 100000 -batchSize 5000

# Or use go run directly
go run . -limit 100000 -batchSize 5000
```

### Command-line flags

- `-data` – Path to the TSV file (default: `data/msmarco-docs.tsv`)
- `-limit` – Maximum number of documents to load (default: 1000, set to 0 for all)
- `-batchSize` – Number of documents to process per batch (default: 1000)

### Search interface

Once the index is ready, you can enter search terms at the prompt. 
Type `>exit` or press `Ctrl+D` to quit.

Example:
```
Search: machine learning algorithms
📊 Found 42 result(s)
...
```

## Performance

Current indexing performance: ~1,760 documents/second
- 250K documents: ~2.5 minutes
- 3M documents: ~28 minutes

See open issues for planned optimizations (parallelization, tokenization improvements).

## Project Layout

- `internal/data` – Batch-based dataset loader with metadata storage
- `internal/text` – Tokenizer with stopword filtering
- `internal/index` – Inverted index builder with incremental updates
- `internal/search` – Query evaluation and TF-IDF ranking
- `internal/output` – CLI rendering helpers
