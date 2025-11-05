![ChatGPT Image Oct 27, 2025 at 10_54_42 AM](https://github.com/user-attachments/assets/13e8d57f-7529-4178-8d87-b95b1179d882)

# luen-search-engine

Go rewrite of the seminar's Python search engine project. 
The implementation keeps the original behaviour: load a slice of the MS MARCO document collection, 
build an inverted index, and offer a small CLI search experience with AND semantics across query terms.

## Prerequisites

- Go 1.22+
- The `msmarco-docs.tsv` [dataset](https://microsoft.github.io/msmarco/Datasets.html#datasets) (same file used by the Python version). Place it anywhere convenient – by default the program looks for `data/msmarco-docs.tsv`.

## Getting Started

A `makefile` is provided for common tasks:

```bash
make help          # Show currently available commands
```

Optional flags:

- `-data` – path to the TSV file if you stored it elsewhere
- `-limit` – number of documents to ingest (default 1000, set to 0 for all rows)

Command to use the limit flag:
`make build ./bin/luen-search-engine -limit 10000`

Once the index is ready you can enter search terms. Use `>exit` (just like the
Python implementation) or press `Ctrl+D` to quit.

## Project Layout

- `internal/data` – dataset loader mirroring the pandas setup
- `internal/text` – simple tokenizer + stopword filter emulating Whoosh's defaults
- `internal/index` – inverted index builder
- `internal/search` – query evaluation and ranking
- `internal/output` – CLI rendering helpers
