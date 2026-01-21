# Build stage
FROM golang:1.23 AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o /pipeline .

# Install Playwright browsers only (deps are in runtime stage)
RUN go install github.com/playwright-community/playwright-go/cmd/playwright@latest && \
    playwright install chromium

# Runtime stage
FROM debian:bookworm-slim

# Install Playwright runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libnss3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    libpango-1.0-0 \
    libcairo2 \
    fonts-liberation \
    fonts-noto-color-emoji \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary and pre-installed browsers
COPY --from=builder /pipeline .
COPY --from=builder /root/.cache/ms-playwright /app/.cache/ms-playwright

ENV PLAYWRIGHT_BROWSERS_PATH=/app/.cache/ms-playwright

# Non-root user
RUN useradd -m -s /bin/bash appuser && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

CMD ["./pipeline"]
