# Go Project Configuration
GO_DIRS = proxy system-metrics page pkg/db pkg/logger pkg/secrets

.PHONY: go-format go-lint go-update go-test go-cov page-build metrics-build proxy-build
.PHONY: _go-format-internal _go-lint-internal _go-update-internal _go-test-internal _go-cov-internal
.PHONY: _page-build-internal _metrics-build-internal _proxy-build-internal

# Public Wrappers (Auto-Nix)
go-format:
	@$(NIX) "make _go-format-internal"

go-lint:
	@$(NIX) "make _go-lint-internal"

go-update:
	@$(NIX) "make _go-update-internal"

go-test:
	@$(NIX) "make _go-test-internal"

go-cov:
	@$(NIX) "make _go-cov-internal"

page-build:
	@$(NIX) "make _page-build-internal"

metrics-build:
	@$(NIX) "make _metrics-build-internal"

proxy-build:
	@$(NIX) "make _proxy-build-internal"

# Internal Implementation
_go-format-internal:
	@echo "Formatting Go code..."
	@gofmt -w -s $(GO_DIRS)

_go-lint-internal:
	@echo "Running go vet (as lint)..."
	@for dir in $(GO_DIRS); do \
		echo "Vetting $$dir..."; \
		(cd $$dir && go vet ./...) || exit 1; \
	done

_go-update-internal:
	@echo "Updating Go dependencies..."
	@for dir in $(GO_DIRS); do \
		echo "Updating $$dir..."; \
		(cd $$dir && go get -u ./... && go mod tidy) || exit 1; \
	done

_go-test-internal:
	@echo "Running Go tests..."
	@for dir in $(GO_DIRS); do \
		echo "Testing $$dir..."; \
		(cd $$dir && go test ./... -v) || exit 1; \
	done

_go-cov-internal:
	@echo "Running tests with coverage..."
	@for dir in $(GO_DIRS); do \
		echo "Coverage for $$dir..."; \
		(cd $$dir && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out) || exit 1; \
	done

_page-build-internal:
	@echo "Running page build..."
	@cd page && go build -o page.exe . && ./page.exe && rm page.exe

_metrics-build-internal:
	@echo "Building system metrics collector..."
	@cd system-metrics && go build -o metrics-collector main.go
	@sudo systemctl restart system-metrics.timer

_proxy-build-internal:
	@echo "Updating Proxy..."
	@cd proxy && go build -o proxy_server .
	@sudo systemctl restart proxy.service
