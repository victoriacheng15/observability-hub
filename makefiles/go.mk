# Go Project Configuration
GO_PACKAGES = ./cmd/... ./internal/...

.PHONY: format test test-cov update vet vuln-scan setup-tailwind web-build proxy-build mcp-build

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

mcp-build:
	@echo "Updating mcp_obs_hub..." && \
	go build -o ./bin/mcp_obs_hub ./cmd/mcp-obs-hub && \
	sudo install -m 755 ./bin/mcp_obs_hub /usr/local/bin/mcp_obs_hub && \
	rm ./bin/mcp_obs_hub
