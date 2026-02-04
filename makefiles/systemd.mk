# Systemd Service Management

# Define exact units to install
ACTIVE_UNITS = proxy.service tailscale-gate.service system-metrics.service \
               system-metrics.timer reading-sync.service reading-sync.timer \
               volume-backup.service volume-backup.timer openbao.service

.PHONY: install-services reload-services uninstall-services bao-status

install-services:
	@echo "🔗 Linking active units..."
	@for unit in $(ACTIVE_UNITS); do \
		sudo ln -sf $(CURDIR)/systemd/$$unit /etc/systemd/system/$$unit; \
	done
	@sudo systemctl daemon-reload
	@echo "🟢 Enabling services..."
	@sudo systemctl enable --now proxy.service tailscale-gate.service openbao.service
	@echo "⏰ Enabling timers..."
	@sudo systemctl enable --now system-metrics.timer reading-sync.timer volume-backup.timer

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
	@nix-shell --run "export BAO_ADDR='http://127.0.0.1:8200' && bao status"

