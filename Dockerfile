FROM golang:1.24-bullseye AS builder

ARG GH_PAT
ENV GOPRIVATE=github.com/trackvision

WORKDIR /app

# Configure git to use PAT for private repos
RUN git config --global url."https://${GH_PAT}@github.com/".insteadOf "https://github.com/"

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o bin/pipeline ./cmd/pipeline/main.go

FROM golang:1.24-alpine

# https://github.com/golang/go/issues/59305
RUN apk add --no-cache gcompat

WORKDIR /root/

COPY --from=builder /app/bin/pipeline .

EXPOSE 80

CMD ["./pipeline"]
