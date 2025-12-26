FROM golang:1.25-alpine

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the root .env file to the current directory (which is /app)
COPY .env .

# Copy go.mod and go.sum from the system-metrics directory
COPY system-metrics/go.mod system-metrics/go.sum ./
RUN go mod download

# Copy the rest of the system-metrics source code
COPY system-metrics/ .

RUN go build -o system-metrics .

CMD ["./system-metrics"]