# Go Project Configuration
GO_PACKAGES = ./cmd/... ./internal/...

.PHONY: format test test-cov update vet vuln-scan setup-tailwind web-build proxy-build ingestion-build mcp-telemetry-build mcp-pods-build mcp-hub-build service-build mcp-build

format:
	@echo "Formatting Go code..." && \
	gofmt -w -s .

test:
	@echo "Running Go tests..." && \
	go test ./internal/... -v

test-cov:
	@echo "Running tests with coverage..." && \
	go test -coverprofile=coverage.out ./internal/... && \
	go tool cover -func=coverage.out && rm coverage.out

update:
	@echo "Updating Go dependencies..." && \
	go get -u ./... && go mod tidy

vet:
	@echo "Running go vet..." && \
	go vet $(GO_PACKAGES)

vuln-scan:
	@echo "Running govulncheck..." && \
	go run golang.org/x/vuln/cmd/govulncheck@latest $(GO_PACKAGES)

setup-tailwind:
	@echo "Downloading tailwind css cli..." && \
	curl -sL https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 -o tailwindcss && \
	chmod +x tailwindcss

web-build: setup-tailwind
	@echo "Generating static site..." && \
	rm -rf dist && \
	mkdir -p dist && \
	go run ./cmd/web && \
	./tailwindcss -i ./internal/web/templates/input.css -o ./dist/styles.css --minify && \
	rm tailwindcss

proxy-build:
	@echo "Updating Proxy..." && \
	go build -o ./bin/proxy_server ./cmd/proxy && \
	sudo install -m 755 ./bin/proxy_server /usr/local/bin/proxy_server && \
	sudo systemctl restart proxy.service && \
	rm ./bin/proxy_server

ingestion-build:
	@echo "Updating Ingestion..." && \
	go build -o ./bin/ingestion ./cmd/ingestion && \
	sudo install -m 755 ./bin/ingestion /usr/local/bin/ingestion && \
	sudo systemctl restart ingestion.service && \
	rm ./bin/ingestion

mcp-telemetry-build:
	@echo "Updating mcp-telemetry..." && \
	go build -o ./bin/mcp_telemetry ./cmd/mcp-telemetry && \
	sudo install -m 755 ./bin/mcp_telemetry /usr/local/bin/mcp_telemetry && \
	rm ./bin/mcp_telemetry

mcp-pods-build:
	@echo "Updating mcp-pods..." && \
	go build -o ./bin/mcp_pods ./cmd/mcp-pods && \
	sudo install -m 755 ./bin/mcp_pods /usr/local/bin/mcp_pods && \
	rm ./bin/mcp_pods

mcp-hub-build:
	@echo "Updating mcp-hub..." && \
	go build -o ./bin/mcp_hub ./cmd/mcp-hub && \
	sudo install -m 755 ./bin/mcp_hub /usr/local/bin/mcp_hub && \
	rm ./bin/mcp_hub

service-build:
	@echo "Building all services..." && \
	make proxy-build && \
	make ingestion-build

mcp-build:
	@echo "Building all MCP Servers..." && \
	make mcp-telemetry-build && \
	make mcp-pods-build && \
	make mcp-hub-build
