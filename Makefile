.PHONY: build run test clean docker-build docker-run check

# Build the application
build:
	go build -o bin/pipeline ./cmd/pipeline

# Run the application
run:
	go run ./cmd/pipeline

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build Docker image
docker-build:
	docker build -t tv-pipelines-timken .

# Run Docker container
docker-run:
	docker run -p 8080:8080 --env-file .env tv-pipelines-timken

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	go vet ./...

# Full check (format, lint, test)
check:
	go fmt ./...
	go vet ./...
	go test -v ./...
