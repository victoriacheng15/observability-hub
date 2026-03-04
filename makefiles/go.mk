# Go Project Configuration
GO_PACKAGES = ./cmd/... ./internal/...

.PHONY: format test test-cov update vet vuln-scan setup-tailwind web-build proxy-build ingestion-build

format:
	$(NIX_WRAP) \
	echo "Formatting Go code..." && \
	gofmt -w -s .

test:
	$(NIX_WRAP) \
	echo "Running Go tests..." && \
	go test $(GO_PACKAGES) -v

test-cov:
	$(NIX_WRAP) \
	echo "Running tests with coverage..." && \
	go test -coverprofile=coverage.out $(GO_PACKAGES) && \
	go tool cover -func=coverage.out && rm coverage.out

update:
	$(NIX_WRAP) \
	echo "Updating Go dependencies..." && \
	go get -u ./... && go mod tidy

vet:
	$(NIX_WRAP) \
	echo "Running go vet..." && \
	go vet $(GO_PACKAGES)

vuln-scan:
	$(NIX_WRAP) \
	echo "Running govulncheck..." && \
	go run golang.org/x/vuln/cmd/govulncheck@latest $(GO_PACKAGES)

setup-tailwind:
	echo "Downloading tailwind css cli v4..." && \
	curl -sL https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 -o tailwindcss && \
	chmod +x tailwindcss

web-build: setup-tailwind
	$(NIX_WRAP) echo "Running web build..." && \
	rm -rf dist && \
	mkdir -p dist && \
	(cd cmd/web && go build -o ../../web-ssg .) && \
	(cd cmd/web && ../../web-ssg) && \
	./tailwindcss -i ./internal/web/templates/input.css -o ./dist/styles.css --minify && \
	rm web-ssg && \
	rm tailwindcss

proxy-build:
	$(NIX_WRAP) \
	echo "Updating Proxy..." && \
	cd cmd/proxy && go build -o ../../bin/proxy_server . && \
	sudo systemctl restart proxy.service

ingestion-build:
	$(NIX_WRAP) \
	echo "Updating ingestion service..." && \
	cd cmd/ingestion && go build -o ../../bin/ingestion . && \
	sudo systemctl restart ingestion.timer
