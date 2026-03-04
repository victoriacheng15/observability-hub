# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy root module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy shared libraries and collectors service
COPY internal/ internal/
COPY cmd/collectors/ cmd/collectors/

# Build the binary from the root module context
RUN CGO_ENABLED=0 GOOS=linux go build -o collectors ./cmd/collectors

# Stage 2: Runtime
FROM alpine:3.20

# Install latest Tailscale CLI to match host version and avoid warnings
RUN apk add --no-cache ca-certificates
RUN wget https://pkgs.tailscale.com/stable/tailscale_latest_amd64.tgz && \
    tar -xzf tailscale_latest_amd64.tgz && \
    cp tailscale_*/tailscale /usr/bin/tailscale && \
    rm -rf tailscale_*

WORKDIR /app
COPY --from=builder /app/collectors .

ENTRYPOINT ["./collectors"]
