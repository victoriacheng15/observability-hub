help:
	@echo "Available commands:"
	@echo "  make create            - Create necessary docker volumes"
	@echo "  make backup            - Backup docker volumes"
	@echo "  make restore           - Restore docker volumes from backup"
	@echo "  make metrics-build     - Build the system metrics collector"
	@echo "  make proxy-up          - Start the go proxy server"
	@echo "  make proxy-down        - Stop the go proxy server"
	@echo "  make proxy-update      - Rebuild and restart the go proxy server"
	@echo "  make promtail-up       - Start the promtail log collector"
	@echo "  make promtail-down     - Stop the promtail log collector"
	@echo "  make promtail-update   - Restart the promtail log collector"

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

# System Metrics Collector
metrics-build:
	@echo "Building system metrics collector..."
	@go build -o metrics-collector.exe main.go

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

# Promtail Management
promtail-up:
	@echo "Starting promtail..."
	@docker run -d \
		--name promtail_server \
		--restart unless-stopped \
		--network host \
		-v $(PWD)/docker/promtail/promtail-config.yaml:/etc/promtail/config.yaml \
		-v /var/lib/docker/containers:/var/lib/docker/containers:ro \
		-v /var/run/docker.sock:/var/run/docker.sock:ro \
		grafana/promtail:latest \
		-config.file=/etc/promtail/config.yaml

promtail-down:
	@echo "Stopping promtail..."
	@docker stop promtail_server || true
	@docker rm promtail_server || true

promtail-update: promtail-down promtail-up
	@echo "Promtail updated."
