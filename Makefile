help:
	@echo "Available commands:"
	@echo "  make rfc                - Create a new RFC (Architecture Decision Record)"
	@echo "  make up                 - Start all docker containers"
	@echo "  make down               - Stop all docker containers"
	@echo "  make create             - Create necessary docker volumes"
	@echo "  make backup             - Backup docker volumes"
	@echo "  make restore            - Restore docker volumes from backup"
	@echo "  make go-format          - Format and simplify Go code"
	@echo "  make go-test            - Run Go tests"
	@echo "  make go-cov             - Run tests with coverage report"
	@echo "  make page-build         - Build the GitHu Page"
	@echo "  make metrics-build      - Build the system metrics collector"
	@echo "  make proxy-up           - Start the go proxy server"
	@echo "  make proxy-down         - Stop the go proxy server"
	@echo "  make proxy-update       - Rebuild and restart the go proxy server"
	@echo "  make install-services   - Install all systemd units from ./systemd"
	@echo "  make reload-services    - Update systemd units (cp + daemon-reload)"
	@echo "  make uninstall-services - Uninstall all systemd units from ./systemd"

# Architecture Decision Record Creation
rfc:
	@./scripts/create_rfc.sh

# Docker Compose Management
up:
	@docker compose up -d

down:
	@docker compose down

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
	@gofmt -w -s ./proxy ./system-metrics ./page ./pkg

go-test:
	@echo "Running Go tests..."
	@cd proxy && go test ./...
	@cd system-metrics && go test ./...
	@cd page && go test ./...
	@cd pkg/db && go test ./...
	@cd pkg/logger && go test ./...

go-cov:
	@echo "Running tests with coverage..."
	@cd proxy && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out
	@cd system-metrics && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out
	@cd page && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out

# GitHub Pages Build
page-build:
	@echo "Running page build..."
	@cd page && go build -o page.exe . && ./page.exe && rm page.exe

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

# Systemd Service Management
install-services:
	@echo "Installing all systemd units as symlinks..."
	@sudo ln -sf $(CURDIR)/systemd/*.service /etc/systemd/system/
	@sudo ln -sf $(CURDIR)/systemd/*.timer /etc/systemd/system/
	@sudo systemctl daemon-reload
	@echo "Enabling regular timers..."
	@for timer in $$(ls systemd/*.timer 2>/dev/null | grep -v "@"); do \
		timer_name=$$(basename $$timer); \
		sudo systemctl enable --now $$timer_name; \
	done
	@echo "Enabling GitOps for repos..."
	@sudo systemctl enable --now gitops-sync@observability-hub.timer
	@sudo systemctl enable --now gitops-sync@mehub.timer
	@echo "Installation complete."

reload-services:
	@echo "Reloading systemd units..."
	@sudo systemctl daemon-reload
	@echo "Configuration reloaded. Changes in ./systemd are active (timers may need restart)."

uninstall-services:
	@echo "Stopping and disabling all project units..."
	# 1. Explicitly handle known instances
	@sudo systemctl disable --now gitops-sync@observability-hub.timer 2>/dev/null || true
	@sudo systemctl disable --now gitops-sync@mehub.timer 2>/dev/null || true
	# 2. Generic cleanup for all units in ./systemd
	@for unit in $$(ls systemd/*.service systemd/*.timer 2>/dev/null); do \
		unit_name=$$(basename $$unit); \
		sudo systemctl stop $$unit_name 2>/dev/null || true; \
		sudo systemctl disable $$unit_name 2>/dev/null || true; \
		sudo rm /etc/systemd/system/$$unit_name 2>/dev/null || true; \
	done
	@sudo systemctl daemon-reload
	@echo "Uninstallation complete. Systemd is clean."
