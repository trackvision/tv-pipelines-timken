.PHONY: build test run clean deps fmt vet lint check setup-hooks docker-build docker-run

# Build the application
build:
	go build -o bin/pipeline ./cmd/pipeline

# Run tests
test:
	go test -v ./...

# Run HTTP API server (default mode)
run: build
	./bin/pipeline

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

# Lint (requires golangci-lint)
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

# Build Docker image
docker-build:
	docker build -t tv-pipelines-timken .

# Run Docker container
docker-run:
	docker run -p 8080:8080 --env-file .env tv-pipelines-timken
