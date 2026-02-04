# Main Entry Point
include makefiles/common.mk
include makefiles/docker.mk
include makefiles/go.mk
include makefiles/systemd.mk
include makefiles/k3s.mk

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make nix-<command>      - Run any make command inside nix-shell (e.g., make nix-go-test)"
	@echo "  make adr                - Create a new ADR (Architecture Decision Record)"
	@echo "  make up                 - Start all docker containers"
	@echo "  make down               - Stop all docker containers"
	@echo "  make create             - Create necessary docker volumes"
	@echo "  make backup             - Backup docker volumes"
	@echo "  make restore            - Restore docker volumes from backup"
	@echo "  make lint               - Lint markdown files"
	@echo "  make go-format          - Format and simplify Go code (Nix-wrapped)"
	@echo "  make go-update          - Update Go dependencies (Nix-wrapped)"
	@echo "  make go-test            - Run Go tests (Nix-wrapped)"
	@echo "  make go-lint            - Run Go lint/vet (Nix-wrapped)"
	@echo "  make go-cov             - Run tests with coverage report (Nix-wrapped)"
	@echo "  make page-build         - Build the GitHub Page (Nix-wrapped)"
	@echo "  make metrics-build      - Build the system metrics collector (Nix-wrapped)"
	@echo "  make proxy-build        - Build and restart the go proxy server (Nix-wrapped)"
	@echo "  make install-services   - Install all systemd units"
	@echo "  make reload-services    - Update systemd units"
	@echo "  make uninstall-services - Uninstall all systemd units"
	@echo "  make bao-status         - Check OpenBao status"
	@echo "  make k3s-alloy-up       - Deploy/Restart Alloy on K3s"
	@echo "  make k3s-loki-up        - Deploy/Restart Loki on K3s"
	@echo "  make k3s-status         - Show K3s namespace status"