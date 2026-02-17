# Go Project Configuration
GO_DIRS = services/proxy services/system-metrics services/second-brain page pkg/db pkg/logger pkg/metrics pkg/secrets pkg/telemetry pkg/brain pkg/env

.PHONY: go-format go-lint go-update go-test go-cov page-build metrics-build proxy-build brain-sync

go-format:
	$(NIX_WRAP)
	@echo "Formatting Go code..."
	@gofmt -w -s $(GO_DIRS)

go-lint:
	$(NIX_WRAP)
	@echo "Running go vet (as lint)..."
	@for dir in $(GO_DIRS); do \
		echo "Vetting $$dir..."; \
		(cd $$dir && go vet ./...) || exit 1; \
	done

go-vuln-scan:
	$(NIX_WRAP)
	@echo "Running govulncheck..."
	@for dir in $(GO_DIRS); do \
		echo "Scanning $$dir..."; \
		(cd $$dir && go run golang.org/x/vuln/cmd/govulncheck@latest ./...) || exit 1; \
	done

go-update:
	$(NIX_WRAP)
	@echo "Updating Go dependencies..."
	@for dir in $(GO_DIRS); do \
		echo "Updating $$dir..."; \
		(cd $$dir && go get -u ./... && go mod tidy) || exit 1; \
	done

go-test:
	$(NIX_WRAP)
	@echo "Running Go tests..."
	@for dir in $(GO_DIRS); do \
		echo "Testing $$dir..."; \
		(cd $$dir && go test ./... -v) || exit 1; \
	done

go-cov:
	$(NIX_WRAP)
	@echo "Running tests with coverage..."
	@for dir in $(GO_DIRS); do \
		echo "Coverage for $$dir..."; \
		(cd $$dir && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out) || exit 1; \
	done

page-build:
	$(NIX_WRAP)
	@echo "Running page build..."
	@cd page && go build -o page.exe . && ./page.exe && rm page.exe

metrics-build:
	$(NIX_WRAP)
	@echo "Building system metrics collector..." && \
	cd services/system-metrics && go build -o ../../metrics-collector main.go && \
	sudo systemctl restart system-metrics.timer

proxy-build:
	$(NIX_WRAP)
	@echo "Updating Proxy..." && \
	cd services/proxy && go build -o ../../proxy_server . && \
	sudo systemctl restart proxy.service

brain-sync:
	$(NIX_WRAP)
	@echo "Running Second Brain Sync..."
	@cd services/second-brain && go run main.go
