# Go Project Configuration
GO_DIRS = web pkg/brain pkg/collectors pkg/db pkg/env pkg/secrets pkg/telemetry services/collectors services/proxy services/ingestion

.PHONY: go-format go-lint go-update go-test go-cov web-build proxy-build brain-sync

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
	rm -rf web/dist && \
	mkdir -p web/dist && \
	(cd web && go build -o ../web-ssg .) && \
	(cd web && ../web-ssg) && \
	./tailwindcss -i ./web/templates/input.css -o ./web/dist/styles.css --minify && \
	rm web-ssg && \
	rm tailwindcss

proxy-build:
	$(NIX_WRAP) \
	echo "Updating Proxy..." && \
	cd services/proxy && go build -o ../../dist/proxy_server . && \
	sudo systemctl restart proxy.service

ingestion-build:
	$(NIX_WRAP) \
	echo "Updating ingestion service..." && \
	cd services/ingestion && go build -o ../../dist/ingestion . && \
	sudo systemctl restart ingestion.timer
