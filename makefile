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

run: build ## Build and run the binary.
	./$(BIN_DIR)/$(BINARY)

clean: ## Remove build artifacts.
	rm -rf $(BIN_DIR)

help: ## Show available make targets.
	@awk -F':.*##' '/^[a-zA-Z][a-zA-Z0-9_-]*:.*##/ {printf "%-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
