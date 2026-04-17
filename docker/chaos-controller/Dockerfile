# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy root module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/chaos-controller/ cmd/chaos-controller/
COPY internal/hardware-sim/ internal/hardware-sim/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o chaos-controller ./cmd/chaos-controller

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && addgroup -S app -g 1000 \
    && adduser -S app -G app -u 1000

WORKDIR /app
COPY --from=builder /app/chaos-controller .

USER 1000:1000

ENTRYPOINT ["./chaos-controller"]
