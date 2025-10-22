.PHONY: all build test test-unit test-integration test-coverage test-race clean install help

# Default target
all: test build

# Build the binary
build:
	@echo "Building clipcat..."
	go build -ldflags="-s -w" -o clipcat clipcat.go
	@echo "✓ Built clipcat"

# Run all tests
test:
	@echo "Running all tests..."
	go test -v -cover

# Run only unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v -run "^Test[^C][^o]" ./...

# Run only integration tests (those starting with TestCollect, TestWrite, TestEndToEnd)
test-integration:
	@echo "Running integration tests..."
	go test -v -run "^Test(Collect|Write|EndToEnd)" ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race

# Run tests with verbose output and show coverage percentage
test-verbose:
	@echo "Running tests verbosely..."
	go test -v -cover -coverprofile=coverage.out
	@echo "\nCoverage summary:"
	go tool cover -func=coverage.out | grep total

# Quick test (no verbose, just pass/fail)
test-quick:
	@go test -cover

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem

# Clean build artifacts and test files
clean:
	@echo "Cleaning..."
	rm -f clipcat coverage.out coverage.html
	@echo "✓ Cleaned"

# Install to ~/.local/bin
install: build
	@echo "Installing to ~/.local/bin..."
	mkdir -p ~/.local/bin
	cp clipcat ~/.local/bin/
	@echo "✓ Installed to ~/.local/bin/clipcat"

# Install to /usr/local/bin (requires sudo)
install-system: build
	@echo "Installing to /usr/local/bin (requires sudo)..."
	sudo cp clipcat /usr/local/bin/
	sudo chmod +x /usr/local/bin/clipcat
	@echo "✓ Installed to /usr/local/bin/clipcat"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Code formatted"

# Run linter
lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	}
	golangci-lint run
	@echo "✓ Linting passed"

# Get dependencies
deps:
	@echo "Getting dependencies..."
	go get github.com/sabhiram/go-gitignore
	go mod tidy
	@echo "✓ Dependencies installed"

# Show test coverage in terminal
coverage-report:
	@go test -coverprofile=coverage.out > /dev/null 2>&1
	@echo "Coverage by file:"
	@go tool cover -func=coverage.out

# Watch for changes and run tests (requires entr)
watch:
	@command -v entr >/dev/null 2>&1 || { \
		echo "entr not found. Install with: sudo apt install entr"; \
		exit 1; \
	}
	@echo "Watching for changes (Ctrl+C to stop)..."
	@ls *.go | entr -c make test-quick

# Development workflow: format, test, build
dev: fmt test build

# CI workflow: lint, test with race detector, coverage
ci: lint test-race test-coverage

# Help target
help:
	@echo "ClipCat Makefile Commands:"
	@echo ""
	@echo "  make build              - Build the binary"
	@echo "  make test               - Run all tests with coverage"
	@echo "  make test-unit          - Run only unit tests"
	@echo "  make test-integration   - Run only integration tests"
	@echo "  make test-coverage      - Generate HTML coverage report"
	@echo "  make test-race          - Run tests with race detector"
	@echo "  make test-verbose       - Run tests with verbose output"
	@echo "  make test-quick         - Quick test run (less output)"
	@echo "  make bench              - Run benchmarks"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make install            - Install to ~/.local/bin"
	@echo "  make install-system     - Install to /usr/local/bin (sudo)"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Run linter"
	@echo "  make deps               - Install dependencies"
	@echo "  make coverage-report    - Show coverage in terminal"
	@echo "  make watch              - Watch files and run tests on change"
	@echo "  make dev                - Format + test + build"
	@echo "  make ci                 - Full CI workflow"
	@echo "  make help               - Show this help"
