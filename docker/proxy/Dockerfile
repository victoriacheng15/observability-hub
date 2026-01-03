FROM golang:1.25-alpine

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the shared package first
COPY pkg/ ./pkg/

# Copy the proxy service
COPY proxy/ ./proxy/

# Copy the root .env if needed (though usually handled by docker-compose)
COPY .env .

WORKDIR /app/proxy

# Download dependencies (now it can find ../pkg/logger)
RUN go mod download

# Build the binary
RUN go build -o /app/proxy_server .

WORKDIR /app
EXPOSE 8085

CMD ["./proxy_server"]