# 1. Build Stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy specific source code directories required for the worker
COPY cmd/worker/ ./cmd/worker/
COPY internal/env/ ./internal/env/
COPY internal/telemetry/ ./internal/telemetry/
COPY internal/worker/ ./internal/worker/
COPY internal/db/ ./internal/db/
COPY internal/secrets/ ./internal/secrets/

# Build the worker binary (statically linked for Linux)
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker

# 2. Runtime Stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /worker .

# Use the binary as the entrypoint
ENTRYPOINT ["/app/worker"]
