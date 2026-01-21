# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -mod=vendor -o /pipeline .

# Runtime stage - use ultra-lightweight headless-shell for fast cold starts
FROM chromedp/headless-shell:latest

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /pipeline .

EXPOSE 8080

CMD ["./pipeline"]
