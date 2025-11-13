#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

GO_BIN="${GO:-go}"
DATASET_PATH="${DATASET_PATH:-data/msmarco-docs-bench-100000.tsv}"
OUTPUT_PATH="${OUTPUT_PATH:-data/msmarco-queries-bench-1000.txt}"

if [[ ! -f "$DATASET_PATH" ]]; then
  echo "Benchmark dataset not found at $DATASET_PATH" >&2
  echo "Create it first, e.g. head -n 100000 data/msmarco-docs.tsv > $DATASET_PATH" >&2
  exit 1
fi

echo "Generating benchmark queries from $DATASET_PATH ..."
"$GO_BIN" run ./cmd/benchqueries -dataset "$DATASET_PATH" -output "$OUTPUT_PATH"
echo "Queries written to $OUTPUT_PATH"
