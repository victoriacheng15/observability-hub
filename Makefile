help:
	@echo "Available commands:"
	@echo "  make rfc               - Create a new RFC (Architecture Decision Record)"
	@echo "  make create            - Create necessary docker volumes"
	@echo "  make backup            - Backup docker volumes"
	@echo "  make restore           - Restore docker volumes from backup"
	@echo "  make go-format         - Format and simplify Go code"
	@echo "  make go-test           - Run Go tests"
	@echo "  make go-cov            - Run tests with coverage report"
	@echo "  make page-build        - Build the GitHu Page"
	@echo "  make metrics-build     - Build the system metrics collector"
	@echo "  make proxy-up          - Start the go proxy server"
	@echo "  make proxy-down        - Stop the go proxy server"
	@echo "  make proxy-update      - Rebuild and restart the go proxy server"

# Architecture Decision Record Creation
rfc:
	@./scripts/create_rfc.sh

# Docker Volume Management
create:
	@echo "Running create volume script..."
	@./scripts/manage_volume.sh create

backup:
	@echo "Running backup volume script..."
	@./scripts/manage_volume.sh backup

restore:
	@echo "Running restore volume script..."
	@./scripts/manage_volume.sh restore

go-format:
	@echo "Formatting Go code..."
	@gofmt -w -s ./proxy ./system-metrics ./page

go-test:
	@echo "Running Go tests..."
	@cd proxy && go test ./...
	@cd system-metrics && go test ./...
	@cd page && go test ./...

go-cov:
	@echo "Running tests with coverage..."
	@cd proxy && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out
	@cd system-metrics && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out
	@cd page && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out

# GitHub Pages Build
page-build:
	@echo "Running page build..."
	@cd page && go build -o page.exe ./main.go && ./page.exe

# System Metrics Collector
metrics-build:
	@echo "Building system metrics collector..."
	@cd system-metrics && go build -o metrics-collector.exe main.go

# Go Proxy Server Management
proxy-up:
	@echo "Starting proxy server..."
	@docker build -t proxy_server -f ./docker/proxy/Dockerfile .
	@docker run -d \
		--name proxy_server \
		--restart unless-stopped \
		--network host \
		proxy_server

proxy-down:
	@echo "Stopping proxy server..."
	@docker stop proxy_server || true
	@docker rm proxy_server || true

proxy-update: proxy-down proxy-up
	@echo "Proxy server updated."
