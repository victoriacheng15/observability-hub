help:
	@echo "Available commands:"
	@echo "  make nix-<command>      - Run any make command inside nix-shell (e.g., make nix-go-test)"
	@echo "  make rfc                - Create a new RFC (Architecture Decision Record)"
	@echo "  make up                 - Start all docker containers"
	@echo "  make down               - Stop all docker containers"
	@echo "  make create             - Create necessary docker volumes"
	@echo "  make backup             - Backup docker volumes"
	@echo "  make restore            - Restore docker volumes from backup"
	@echo "  make go-format          - Format and simplify Go code"
	@echo "  make go-update          - Update Go dependencies (go get -u && go mod tidy)"
	@echo "  make go-test            - Run Go tests"
	@echo "  make go-cov             - Run tests with coverage report"
	@echo "  make page-build         - Build the GitHu Page"
	@echo "  make metrics-build      - Build the system metrics collector"
	@echo "  make proxy-build        - Build and restart the go proxy server"
	@echo "  make install-services   - Install all systemd units from ./systemd"
	@echo "  make reload-services    - Update systemd units (cp + daemon-reload)"
	@echo "  make uninstall-services - Uninstall all systemd units from ./systemd"

# Run any target inside nix-shell
nix-%:
	@nix-shell --run "make $*"

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

# Go Project Configuration
GO_DIRS = proxy system-metrics page pkg/db pkg/logger pkg/secrets

go-format:
	@echo "Formatting Go code..."
	@gofmt -w -s $(GO_DIRS)

go-vet:
	@echo "Running go vet..."
	@for dir in $(GO_DIRS); do \
		echo "Vetting $$dir..."; \
		(cd $$dir && go vet ./...); \
	done

go-update:
	@echo "Updating Go dependencies..."
	@for dir in $(GO_DIRS); do \
		echo "Updating $$dir..."; \
		(cd $$dir && go get -u ./... && go mod tidy); \
	done

go-test:
	@echo "Running Go tests..."
	@for dir in $(GO_DIRS); do \
		echo "Testing $$dir..."; \
		(cd $$dir && go test ./... -v); \
	done

go-cov:
	@echo "Running tests with coverage..."
	@for dir in $(GO_DIRS); do \
		echo "Coverage for $$dir..."; \
		(cd $$dir && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out); \
	done

# GitHub Pages Build
page-build:
	@echo "Running page build..."
	@cd page && go build -o page.exe . && ./page.exe && rm page.exe

# System Metrics Collector
metrics-build:
	@echo "Building system metrics collector..."
	@cd system-metrics && go build -o metrics-collector main.go
	@sudo systemctl restart system-metrics.timer

# Go Proxy Server Management
proxy-build:
	@echo "Updating Proxy..."
	@cd proxy && go build -o proxy_server .
	@sudo systemctl restart proxy.service

# Systemd Service Management

# Define exact units to install
ACTIVE_UNITS = proxy.service tailscale-gate.service system-metrics.service \
               system-metrics.timer reading-sync.service reading-sync.timer \
               volume-backup.service volume-backup.timer openbao.service

install-services:
	@echo "🔗 Linking active units..."
	@for unit in $(ACTIVE_UNITS); do \
		sudo ln -sf $(CURDIR)/systemd/$$unit /etc/systemd/system/$$unit; \
	done
	@sudo systemctl daemon-reload
	@echo "🟢 Enabling services..."
	@sudo systemctl enable --now proxy.service tailscale-gate.service openbao.service
	@echo "⏰ Enabling timers..."
	@sudo systemctl enable --now system-metrics.timer reading-sync.timer volume-backup.timer

reload-services:
	@echo "Reloading systemd units..."
	@sudo systemctl daemon-reload
	@echo "Configuration reloaded. Changes in ./systemd are active (timers may need restart)."

# OpenBao Management
bao-status:
	@nix-shell --run "export BAO_ADDR='http://127.0.0.1:8200' && bao status"

uninstall-services:
	@echo "🛑 Nuclear Cleanup: Stopping and removing all project units..."
	@for unit in $$(ls systemd/ 2>/dev/null); do \
		sudo systemctl disable --now $$unit 2>/dev/null || true; \
		sudo rm /etc/systemd/system/$$unit 2>/dev/null || true; \
	done
	@sudo systemctl daemon-reload
	@echo "🧹 Systemd is clean."
