# gcpclosecheck Makefile
# Quality assurance and performance optimization tasks

.PHONY: help build test test-short test-e2e test-integration bench lint vet fmt clean install deps quality ci
.DEFAULT_GOAL := help

# Build variables
BINARY_NAME=gcpclosecheck
MAIN_PACKAGE=./cmd/gcpclosecheck
BUILD_DIR=./bin
VERSION?=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

help: ## Show this help message
	@echo "gcpclosecheck - GCP resource close/cancel detection linter"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Dependencies
deps: ## Install dependencies
	go mod download
	go mod tidy
	go mod verify

# Build
build: deps ## Build the binary
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

install: deps ## Install to GOPATH/bin
	go install $(LDFLAGS) $(MAIN_PACKAGE)

# Testing
test: ## Run all tests
	go test -v ./...

test-short: ## Run tests in short mode
	go test -short -v ./...

test-e2e: ## Run E2E golden tests
	go test -v ./internal/analyzer -run TestE2E

test-integration: ## Run integration tests  
	go test -v ./cmd/gcpclosecheck -run TestIntegration

test-coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Benchmarking
bench: ## Run benchmarks
	go test -bench=. -benchmem -v ./internal/analyzer

bench-cpu: ## Run CPU profiling benchmarks
	go test -bench=. -benchmem -cpuprofile=cpu.prof -v ./internal/analyzer
	@echo "CPU profile generated: cpu.prof"

bench-mem: ## Run memory profiling benchmarks
	go test -bench=. -benchmem -memprofile=mem.prof -v ./internal/analyzer
	@echo "Memory profile generated: mem.prof"

# Code quality
fmt: ## Format Go code
	go fmt ./...
	goimports -w .

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	golangci-lint run

lint-fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix

# Quality assurance
quality: fmt vet lint test ## Run all quality checks

# Security
security: ## Run security checks
	gosec ./...

# CI/CD pipeline
ci: deps quality test-coverage bench ## Full CI pipeline

# Performance optimization
perf-test: ## Run performance tests
	@echo "Running performance optimization tests..."
	go test -bench=BenchmarkAnalyzer -benchmem -v ./internal/analyzer
	go test -bench=BenchmarkMemoryUsage -benchmem -v ./internal/analyzer
	go test -bench=BenchmarkConcurrentAnalysis -benchmem -v ./internal/analyzer

memory-test: ## Run memory usage tests
	@echo "Testing memory usage..."
	go test -run=TestE2EMemoryUsage -v ./internal/analyzer
	go test -bench=BenchmarkMemoryUsage -benchmem -memprofile=mem.prof -v ./internal/analyzer

# Performance analysis
prof-cpu: bench-cpu ## Analyze CPU profile
	go tool pprof cpu.prof

prof-mem: bench-mem ## Analyze memory profile
	go tool pprof mem.prof

# Integration with go vet
vet-tool: build ## Test as go vet tool
	go vet -vettool=$(BUILD_DIR)/$(BINARY_NAME) ./testdata/...

# Docker support
docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME):$(VERSION) .

docker-test: ## Run tests in Docker
	docker run --rm -v $(PWD):/workspace -w /workspace golang:1.21 make test

# Clean
clean: ## Clean build artifacts and temporary files
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -f cpu.prof mem.prof
	go clean -cache
	go clean -testcache
	go clean -modcache

# Release
release-dry: ## Dry run release
	goreleaser release --snapshot --rm-dist

release: ## Create release
	goreleaser release

# Development helpers
dev-setup: ## Setup development environment
	@echo "Setting up development environment..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/goreleaser/goreleaser@latest
	@echo "Development environment setup complete!"

# Check versions
version: ## Show version information
	@echo "gcpclosecheck version: $(VERSION)"
	@echo "Go version: $(shell go version)"
	@echo "golangci-lint version: $(shell golangci-lint version 2>/dev/null || echo 'not installed')"

# Performance monitoring
monitor: ## Monitor performance during test run
	@echo "Monitoring performance..."
	time make test
	@echo "Performance monitoring complete"

# Memory leak detection
leak-test: ## Test for memory leaks
	go test -race -v ./...

# Static analysis
static-analysis: vet lint security ## Run all static analysis tools

# Full quality gate (for CI/CD)
quality-gate: static-analysis test-coverage bench ## Full quality gate check
	@echo "All quality gates passed!"

# Help for specific targets
test-help: ## Show testing help
	@echo "Testing targets:"
	@echo "  test         - Run all tests"
	@echo "  test-short   - Run tests in short mode"  
	@echo "  test-e2e     - Run E2E golden tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  bench        - Run benchmarks"
	@echo "  memory-test  - Test memory usage"
	@echo "  leak-test    - Test for memory leaks"