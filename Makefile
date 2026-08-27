BINARY := uptime-go
CONFIG ?= configs/uptime.yml
GO ?= go

.PHONY: help build run report test benchmark fmt vet tidy clean

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Compile the application
	$(GO) build -trimpath -o $(BINARY) .

run: build ## Build and run using CONFIG
	./$(BINARY) --config $(CONFIG)

report: build ## Build and show the uptime report
	./$(BINARY) report

test: ## Run all tests
	$(GO) test ./...

benchmark: ## Run all benchmarks
	$(GO) test -run '^$$' -bench . ./benchmark

fmt: ## Format Go source files
	$(GO) fmt ./...

vet: ## Run Go static analysis
	$(GO) vet ./...

tidy: ## Synchronize module dependencies
	$(GO) mod tidy

clean: ## Remove the compiled binary
	$(GO) clean
	$(RM) $(BINARY)
