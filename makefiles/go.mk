# Go Project Configuration
GO_DIRS = web pkg/brain pkg/collectors pkg/db pkg/env pkg/secrets pkg/telemetry services/collectors services/proxy services/reading-sync services/second-brain

.PHONY: go-format go-lint go-update go-test go-cov web-build metrics-build reading-build proxy-build brain-sync setup-tailwind

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

reading-build:
	$(NIX_WRAP) \
	echo "Building reading sync..." && \
	cd services/reading-sync && go build -o ../../dist/reading-sync . && \
	sudo systemctl restart reading-sync.timer

brain-sync:
	$(NIX_WRAP) \
	echo "Running Second Brain Sync..." && \
	cd services/second-brain && go run .

metrics-build:
	$(NIX_WRAP) \
	echo "Building system metrics collector..." && \
	cd services/system-metrics && go build -o ../../dist/metrics-collector . && \
	sudo systemctl restart system-metrics.timer