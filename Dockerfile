# syntax=docker/dockerfile:1
FROM golang:1.24-bullseye AS builder

ENV GOPRIVATE=github.com/trackvision

WORKDIR /app

# Configure git to use SSH for private repos
RUN mkdir -p -m 0700 ~/.ssh && ssh-keyscan github.com >> ~/.ssh/known_hosts
RUN git config --global url."git@github.com:".insteadOf "https://github.com/"

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
