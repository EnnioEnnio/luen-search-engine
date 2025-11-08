#!/usr/bin/env bash
set -euo pipefail

# Helper to strip the leading "D" from MS MARCO doc IDs so they can be stored as integers.
# Note: this preprocessing step might take some time (10 minutes on M1 Pro) on the full corpus.

INPUT_PATH="${1:-data/msmarco-docs.tsv}"
OUTPUT_PATH="${2:-data/msmarco-docs-preprocessed.tsv}"

echo "Preprocessing ${INPUT_PATH} -> ${OUTPUT_PATH}"

awk -F $'\t' 'BEGIN { OFS = "\t" } { $1 = substr($1, 2); print }' "${INPUT_PATH}" > "${OUTPUT_PATH}"

echo "Done."
