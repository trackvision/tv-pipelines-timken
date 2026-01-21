.PHONY: build test run run-once clean deps fmt lint vet check setup-hooks

# Build the unified pipeline binary
build:
	go build -o bin/pipeline ./cmd/pipeline

# Run tests
test:
	go test -v ./...

# Run HTTP API server (default mode)
run: build
	./bin/pipeline

# Run COC pipeline once via CLI
run-once: build
	./bin/pipeline --run-once --pipeline=coc --sscc=$(SSCC)

# List available pipelines
list: build
	./bin/pipeline --list

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
deps:
	go mod tidy
	go mod download

# Format code
fmt:
	go fmt ./...

# Static analysis
vet:
	go vet ./...

# Lint
lint:
	golangci-lint run ./...

# Run all checks (vet, lint, test)
check: vet lint test

# Install git hooks for pre-push checks
setup-hooks:
	@echo "Installing pre-push hook..."
	@cp scripts/pre-push .git/hooks/pre-push
	@chmod +x .git/hooks/pre-push
	@echo "Pre-push hook installed successfully!"
	@echo "Hook will run: go vet, golangci-lint, go test before each push"
