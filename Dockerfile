# syntax=docker/dockerfile:1
FROM golang:1.24-bullseye AS builder

ENV GOPRIVATE=github.com/trackvision

WORKDIR /app

# Set env for private repo
RUN mkdir -p -m 0600 ~/.ssh \
    && echo "Host github.com\n\tStrictHostKeyChecking no\n" >> ~/.ssh/config \
    && git config --global url."ssh://git@github.com/".insteadOf https://github.com/

COPY go.mod go.sum ./

RUN --mount=type=ssh go mod download

COPY . .

RUN go build -o bin/pipeline ./cmd/pipeline/main.go

FROM golang:1.24-alpine

# https://github.com/golang/go/issues/59305
RUN apk add --no-cache gcompat

WORKDIR /root/

COPY --from=builder /app/bin/pipeline .

EXPOSE 80

CMD ["./pipeline"]
