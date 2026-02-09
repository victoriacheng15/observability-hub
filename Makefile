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
	@echo "  make adr                - Create a new ADR (Architecture Decision Record)"
	@echo "  make lint               - Lint markdown files"
	@echo ""
	@echo "Go Development (Auto-Nix):"
	@echo "  make go-format          - Format and simplify Go code"
	@echo "  make go-update          - Update Go dependencies"
	@echo "  make go-test            - Run Go tests"
	@echo "  make go-lint            - Run Go lint/vet"
	@echo "  make go-cov             - Run tests with coverage report"
	@echo "  make page-build         - Build the GitHub Page"
	@echo "  make metrics-build      - Build the system metrics collector"
	@echo "  make proxy-build        - Build and restart the go proxy server"
	@echo "  make brain-sync         - Run the second brain knowledge ingestion"
	@echo ""
	@echo "Host Tier (Systemd & Secrets):"
	@echo "  make install-services   - Install all systemd units"
	@echo "  make reload-services    - Update systemd units"
	@echo "  make uninstall-services - Uninstall all systemd units"
	@echo "  make bao-status         - Check OpenBao status"
	@echo ""
	@echo "Kubernetes Tier (k3s):"
	@echo "  make k3s-status         - Show K3s namespace status"
	@echo "  make k3s-df             - Check cluster image disk usage"
	@echo "  make k3s-prune          - Cleanup unused images and ghost pods"
	@echo "  make k3s-alloy-up       - Deploy/Restart Alloy"
	@echo "  make k3s-loki-up        - Deploy/Restart Loki"
	@echo "  make k3s-grafana-up     - Deploy/Restart Grafana"
	@echo "  make k3s-postgres-up    - Deploy/Restart PostgreSQL"
	@echo "  make k3s-backup-<name>  - Backup a resource (e.g., make k3s-backup-postgres)"