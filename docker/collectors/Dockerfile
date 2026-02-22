# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files (libraries and service)
COPY pkg/collectors/ pkg/collectors/
COPY pkg/db/ pkg/db/
COPY pkg/env/ pkg/env/
COPY pkg/secrets/ pkg/secrets/
COPY pkg/telemetry/ pkg/telemetry/
COPY services/collectors/ services/collectors/

# Tidy and build
WORKDIR /app/services/collectors
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o collectors .

# Stage 2: Runtime
FROM alpine:3.20

# Install latest Tailscale CLI to match host version and avoid warnings
RUN apk add --no-cache ca-certificates
RUN wget https://pkgs.tailscale.com/stable/tailscale_latest_amd64.tgz && \
    tar -xzf tailscale_latest_amd64.tgz && \
    cp tailscale_*/tailscale /usr/bin/tailscale && \
    rm -rf tailscale_*

WORKDIR /app
COPY --from=builder /app/services/collectors/collectors .

ENTRYPOINT ["./collectors"]
