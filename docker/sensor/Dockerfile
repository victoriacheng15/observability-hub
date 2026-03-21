# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy root module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy sensor source
COPY cmd/sensor/ cmd/sensor/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o sensor ./cmd/sensor

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/sensor .

ENTRYPOINT ["./sensor"]
