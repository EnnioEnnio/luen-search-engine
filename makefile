.PHONY: build test lint format tidy run clean help

GO ?= go
BIN_DIR := bin
BINARY := luen

build: ## Build the project binary.
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BINARY) .

test: ## Run all Go tests.
	$(GO) test ./...

lint: ## Run static analysis checks.
	$(GO) vet ./...

format: ## Format all Go source files.
	$(GO) fmt ./...

tidy: ## Ensure go.mod and go.sum are up to date.
	$(GO) mod tidy

dev: build ## Build and run the binary with a smaller corpus (1000).
	./$(BIN_DIR)/$(BINARY)

run: build ## Build and run the binary with a larger corpus (15000).
	./$(BIN_DIR)/$(BINARY) -limit 15000

clean: ## Remove build artifacts.
	rm -rf $(BIN_DIR)

help: ## Show available make targets.
	@awk -F':.*##' '/^[a-zA-Z][a-zA-Z0-9_-]*:.*##/ {printf "%-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
