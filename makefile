.PHONY: build test lint format tidy run clean help bench

GO ?= go
BIN_DIR := bin
BINARY := luen
INDEX_DIR := index
EMBED_DIR := embedding
EMBED_OUTPUT := output

build: ## Build the project binary.
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BINARY) .

test: ## Run all Go tests.
	$(GO) test ./... -v

lint: setup-python ## Run static analysis checks.
	$(GO) vet ./...
	cd embedding && uv run ruff check .

format: setup-python ## Format all Go source files.
	$(GO) fmt ./...
	cd embedding && uv run ruff format .

tidy: ## Ensure go.mod and go.sum are up to date.
	$(GO) mod tidy

index: build ## Build the inverted index with all documents
	./$(BIN_DIR)/$(BINARY) -buildindex -limit 0

dev: build ## Build and run the binary with a smaller corpus (10000).
	./$(BIN_DIR)/$(BINARY) -limit 10000

run: build ## Build and run the binary using the on-disk index (auto-builds 100k docs if missing).
	./$(BIN_DIR)/$(BINARY) -disklimit 100000

start: build ## Start the HTTP server using the on-disk index.
	./$(BIN_DIR)/$(BINARY) -server

frontend: ## Compile TypeScript frontend.
	npx -y -p typescript tsc

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
	cd embedding && uv run python create_embedding.py --data ../data/msmarco-docs-preprocessed.tsv --output-dir $(EMBED_OUTPUT) --batch-size 10

serve-embedding: setup-python ## Run the semantic search gRPC service.
	cd embedding && uv run python semantic_search.py

setup-python: ## Install python dependencies using uv.
	cd embedding && uv sync

clean: ## Remove build artifacts.
	rm -rf $(BIN_DIR)
	rm -rf $(INDEX_DIR)
	rm -rf $(EMBED_DIR)/$(EMBED_OUTPUT)

help: ## Show available make targets.
	@awk -F':.*##' '/^[a-zA-Z][a-zA-Z0-9_-]*:.*##/ {printf "%-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
