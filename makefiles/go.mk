# Go Project Configuration
GO_DIRS = cmd/web cmd/proxy cmd/collectors cmd/ingestion internal/web internal/brain internal/collectors internal/db internal/env internal/secrets internal/telemetry

.PHONY: go-format go-lint go-vuln-scan go-update go-test go-test-cov setup-tailwind web-build proxy-build ingestion-build

go-format:
	$(NIX_WRAP) \
	echo "Formatting Go code..." && \
	gofmt -w -s $(GO_DIRS)

go-lint:
	$(NIX_WRAP) \
	echo "Running go vet (as lint)..." && \
	for dir in $(GO_DIRS); do \
		echo "Vetting $$dir..."; \
		(cd $$dir && go vet ./...) || exit 1; \
	done

go-vuln-scan:
	$(NIX_WRAP) \
	echo "Running govulncheck..." && \
	for dir in $(GO_DIRS); do \
		echo "Scanning $$dir..."; \
		(cd $$dir && go run golang.org/x/vuln/cmd/govulncheck@latest ./...) || exit 1; \
	done

go-update:
	$(NIX_WRAP) \
	echo "Updating Go dependencies..." && \
	for dir in $(GO_DIRS); do \
		echo "Updating $$dir..."; \
		(cd $$dir && go get -u ./... && go mod tidy) || exit 1; \
	done

go-test:
	$(NIX_WRAP) \
	echo "Running Go tests..." && \
	for dir in $(GO_DIRS); do \
		echo "Testing $$dir..."; \
		(cd $$dir && go test ./... -v) || exit 1; \
	done

go-test-cov:
	$(NIX_WRAP) \
	echo "Running tests with coverage..." && \
	for dir in $(GO_DIRS); do \
		echo "Coverage for $$dir..."; \
		(cd $$dir && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out && rm coverage.out) || exit 1; \
	done

setup-tailwind:
	echo "Downloading tailwind css cli v4..." && \
	curl -sL https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 -o tailwindcss && \
	chmod +x tailwindcss

web-build: setup-tailwind
	$(NIX_WRAP) echo "Running web build..." && \
	rm -rf cmd/web/dist && \
	mkdir -p cmd/web/dist && \
	(cd cmd/web && go build -o ../../web-ssg .) && \
	(cd cmd/web && ../../web-ssg) && \
	./tailwindcss -i ./internal/web/templates/input.css -o ./cmd/web/dist/styles.css --minify && \
	rm web-ssg && \
	rm tailwindcss

proxy-build:
	$(NIX_WRAP) \
	echo "Updating Proxy..." && \
	cd cmd/proxy && go build -o ../../dist/proxy_server . && \
	sudo systemctl restart proxy.service

ingestion-build:
	$(NIX_WRAP) \
	echo "Updating ingestion service..." && \
	cd cmd/ingestion && go build -o ../../dist/ingestion . && \
	sudo systemctl restart ingestion.timer
