# Systemd Service Management

# Define exact units to install
ACTIVE_UNITS = proxy.service tailscale-gate.service openbao.service

.PHONY: install-services reload-services uninstall-services bao-status

install-services:
	@echo "📦 Installing active units..."
	@for unit in $(ACTIVE_UNITS); do \
		sudo rm -f /etc/systemd/system/$$unit; \
		sudo cp $(CURDIR)/systemd/$$unit /etc/systemd/system/$$unit; \
		sudo chmod 644 /etc/systemd/system/$$unit; \
	done
	@sudo systemctl daemon-reload
	@echo "🟢 Enabling services..."
	@sudo systemctl enable --now proxy.service tailscale-gate.service openbao.service

reload-services:
	@echo "Reloading systemd units..."
	@sudo systemctl daemon-reload
	@echo "Configuration reloaded. Changes in ./systemd are active (timers may need restart)."

uninstall-services:
	@echo "🛑 Nuclear Cleanup: Stopping and removing all project units..."
	@for unit in $$(ls systemd/ 2>/dev/null); do \
		sudo systemctl disable --now $$unit 2>/dev/null || true; \
		sudo rm /etc/systemd/system/$$unit 2>/dev/null || true; \
	done
	@sudo systemctl daemon-reload
	@echo "🧹 Systemd is clean."

# OpenBao Management
bao-status:
	$(NIX_WRAP)
	@export BAO_ADDR='http://127.0.0.1:8200' && bao status

