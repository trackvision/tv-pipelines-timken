.PHONY: build test run run-once clean deps fmt lint

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

# Lint
lint:
	golangci-lint run
