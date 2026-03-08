# Main Entry Point
include makefiles/common.mk
include makefiles/go.mk
include makefiles/systemd.mk
include makefiles/k3s.mk

.PHONY: help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Project Management:"
	@echo "  make adr                  - Create a new ADR (Architecture Decision Record)"
	@echo "  make lint                 - Lint markdown files"
	@echo "  make lint-configs         - Lint HCL policies and GitHub Actions"
	@echo ""
	@echo "Go Development (Auto-Nix):"
	@echo "  make format               - Format and simplify Go code"
	@echo "  make test                 - Run Go tests"
	@echo "  make test-cov             - Run tests with coverage report"
	@echo "  make update               - Update Go dependencies"
	@echo "  make vet                  - Run Go lint/vet"
	@echo "  make vuln-scan            - Run govulncheck for security vulnerabilities"
	@echo "  make web-build            - Build the GitHub Page"
	@echo "  make proxy-build          - Build and restart the go proxy server"
	@echo "  make ingestion-build      - Build and restart the go ingestion server"
	@echo "  make mcp-telemetry-build  - Build and restart the go mcp telemetry server"
	@echo "	 make all-build            - Build and restart all Go services"
	@echo ""
	@echo "Host Tier (Systemd & Secrets):"
	@echo "  make install-services    - Install all systemd units"
	@echo "  make reload-services     - Update systemd units"
	@echo "  make uninstall-services  - Uninstall all systemd units"
	@echo "  make bao-status          - Check OpenBao status"
	@echo ""
	@echo "Kubernetes Tier (k3s):"
	@echo "  make kube-lint           - Lint Kubernetes manifests"
	@echo "  make k3s-status          - Show K3s namespace status"
	@echo "  make k3s-df              - Check cluster image disk usage"
	@echo "  make k3s-prune           - Cleanup unused images and ghost pods"
	@echo "  make k3s-collectors-up   - Deploy/Restart Collectors"
	@echo "  make k3s-logs-<pod>      - Tail logs for a specific pod (e.g., make k3s-logs-loki-0)"
	@echo "  make k3s-backup-<name>   - Backup a resource (e.g., make k3s-backup-postgres)"
