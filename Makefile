BINARY_NAME=consul-review
MAIN_PATH=./main.go

# Build variables (for go-build-release)
BUILD_VERSION ?= dev
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_BY ?= manual

export GITHUB_REPOSITORY_OWNER ?= binsabbar

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help go-build go-build-release go-test go-test-verbose go-test-coverage go-lint go-vulncheck go-fmt go-fmt-check release release-snapshot clean

help:
	@echo "$(BLUE)consul-review - Makefile Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)Development:$(NC)"
	@echo "  make go-build              - Build the binary to bin/$(BINARY_NAME)"
	@echo "  make go-build-release      - Build with version info (mimics GoReleaser)"
	@echo "  make go-test               - Run all tests with race detector"
	@echo "  make go-test-verbose       - Run all tests with verbose output"
	@echo "  make go-test-coverage      - Run all tests with race detector and coverage"
	@echo "  make go-lint               - Run golangci-lint checks"
	@echo "  make go-vulncheck          - Run govulncheck for security vulnerabilities"
	@echo "  make go-fmt                - Format code with gofmt"
	@echo "  make go-fmt-check          - Check if code is properly formatted"
	@echo ""
	@echo "$(YELLOW)Maintenance:$(NC)"
	@echo "  make clean                 - Clean build artifacts (bin/, dist/)"
	@echo ""
	@echo "$(YELLOW)Release:$(NC)"
	@echo "  make release               - Create a tagged release with GoReleaser (requires git tag)"
	@echo "  make release-snapshot      - Build snapshot release locally (no tag required)"
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make go-build-release BUILD_VERSION=1.0.0-alpha.1"
	@echo "  make release-snapshot"
	@echo "  git tag v0.1.0 && make release"

## ─── Go ──────────────────────────────────────────────────────────────────────

go-build:
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p bin
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Built: bin/$(BINARY_NAME)$(NC)"

go-build-release:
	@echo "$(BLUE)Building $(BINARY_NAME) with version info...$(NC)"
	@mkdir -p bin
	go build \
		-ldflags="-s -w \
			-X main.version=$(BUILD_VERSION) \
			-X main.commit=$(BUILD_COMMIT) \
			-X main.date=$(BUILD_DATE) \
			-X main.builtBy=$(BUILD_BY)" \
		-o bin/$(BINARY_NAME) \
		$(MAIN_PATH)
	@echo "$(GREEN)✓ Built: bin/$(BINARY_NAME) (version=$(BUILD_VERSION) commit=$(BUILD_COMMIT))$(NC)"

go-test:
	@echo "$(BLUE)Running tests (with race detector)...$(NC)"
	go test -race ./... -count=1
	@echo "$(GREEN)✓ Tests passed$(NC)"

go-test-verbose:
	@echo "$(BLUE)Running tests (verbose)...$(NC)"
	go test -v ./... -count=1

go-test-coverage:
	@echo "$(BLUE)Running tests with race detector and coverage...$(NC)"
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report: coverage.html$(NC)"

go-lint:
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
	golangci-lint run ./...
	@echo "$(GREEN)✓ Lint passed$(NC)"

go-vulncheck:
	@echo "$(BLUE)Running govulncheck...$(NC)"
	@command -v govulncheck > /dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

go-fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	gofmt -w .
	@echo "$(GREEN)✓ Code formatted$(NC)"

go-fmt-check:
	@echo "$(BLUE)Checking code formatting...$(NC)"
	@UNFORMATTED=$$(gofmt -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "$(RED)✗ Unformatted files:$(NC)"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi
	@echo "$(GREEN)✓ All files properly formatted$(NC)"

## ─── Release ─────────────────────────────────────────────────────────────────

release:
	@echo "$(BLUE)Creating release with GoReleaser...$(NC)"
	goreleaser release --clean

release-snapshot:
	@echo "$(BLUE)Building snapshot release...$(NC)"
	goreleaser release --snapshot --clean

## ─── Maintenance ─────────────────────────────────────────────────────────────

clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf bin/ dist/ coverage.out coverage.html
	@echo "$(GREEN)✓ Clean$(NC)"
