.PHONY: build test lint format tidy run clean help bench

GO ?= go
BIN_DIR := bin
BINARY := luen
INDEX_DIR := index

build: ## Build the project binary.
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BINARY) .

test: ## Run all Go tests.
	$(GO) test ./... -v

lint: ## Run static analysis checks.
	$(GO) vet ./...

format: ## Format all Go source files.
	$(GO) fmt ./...

tidy: ## Ensure go.mod and go.sum are up to date.
	$(GO) mod tidy

index: build ## Build the inverted index with all documents
	./$(BIN_DIR)/$(BINARY) -buildindex -limit 0

dev: build ## Build and run the binary with a smaller corpus (10000).
	./$(BIN_DIR)/$(BINARY) -limit 10000

run: build ## Build and run the binary using the on-disk index (auto-builds 100k docs if missing).
	./$(BIN_DIR)/$(BINARY) -disklimit 100000

bench: ## Run deterministic index + query benchmarks against the 100k MS MARCO subset.
	@if [ ! -f data/msmarco-docs-bench-100000.tsv ]; then \
		echo "Missing benchmark dataset at data/msmarco-docs-bench-100000.tsv"; \
		echo "Create it with: head -n 100000 data/msmarco-docs-preprocessed.tsv > data/msmarco-docs-bench-100000.tsv"; \
		exit 1; \
	fi
	@if [ ! -f data/msmarco-queries-bench-1000.txt ]; then \
		echo "Missing benchmark queries file at data/msmarco-queries-bench-1000.txt"; \
		echo "Generate it with: scripts/generate-bench-queries.sh"; \
		exit 1; \
	fi
	$(GO) test ./internal/index ./internal/indexer ./internal/search -run=^$$ -bench=. -benchmem

embed: setup-python ## Generate embeddings for the dataset using the python script.
	cd embedding && uv run python create_embedding.py --data ../data/msmarco-docs-preprocessed.tsv --output-dir output --batch-size 10

setup-python: ## Install python dependencies using uv.
	cd embedding && uv sync


clean: ## Remove build artifacts.
	rm -rf $(BIN_DIR)
	rm -rf $(INDEX_DIR)

help: ## Show available make targets.
	@awk -F':.*##' '/^[a-zA-Z][a-zA-Z0-9_-]*:.*##/ {printf "%-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
