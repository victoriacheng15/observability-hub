# K3s Orchestration & Migration

.PHONY: k3s-alloy-up k3s-loki-up k3s-tempo-up k3s-opentelemetry-up k3s-prometheus-up k3s-status k3s-df k3s-prune k3s-logs-% k3s-backup-%

# Maintenance
k3s-df:
	@echo "Checking K3s Container Images Usage..."
	@sudo k3s crictl images

k3s-prune:
	@echo "Pruning unused K3s images..."
	@sudo k3s crictl rmi --prune
	@echo "Deleting completed/failed pods across all namespaces..."
	@$(KC) get pods --all-namespaces --field-selector 'status.phase==Succeeded' -o json | jq -r '.items[] | "--namespace=" + .metadata.namespace + " " + .metadata.name' | xargs -r -L1 $(KC) delete pod
	@$(KC) get pods --all-namespaces --field-selector 'status.phase==Failed' -o json | jq -r '.items[] | "--namespace=" + .metadata.namespace + " " + .metadata.name' | xargs -r -L1 $(KC) delete pod

# Apply manifests and rollout restart
k3s-alloy-up:
	@echo "Deploying Alloy..."
	@$(KC) apply -f k3s/alloy/manifest.yaml
	@$(KC) rollout restart daemonset/alloy

k3s-loki-up:
	@echo "Deploying Loki..."
	@$(KC) apply -f k3s/loki/manifest.yaml
	@$(KC) rollout restart statefulset/loki

k3s-tempo-up:
	@echo "Deploying Tempo..."
	@$(KC) apply -f k3s/tempo/manifest.yaml
	@$(KC) rollout restart statefulset/tempo

k3s-opentelemetry-up:
	@echo "Deploying OpenTelemetry Collector..."
	@$(KC) apply -f k3s/opentelemetry/manifest.yaml
	@$(KC) rollout restart deployment/opentelemetry

k3s-prometheus-up:
	@echo "Deploying Prometheus..."
	@$(KC) apply -f k3s/prometheus/manifest.yaml
	@$(KC) rollout restart deployment/prometheus-server

k3s-grafana-up:
	@echo "Deploying Grafana..."
	@$(KC) create configmap grafana-dashboards --namespace $(NS) --from-file=k3s/grafana/dashboards/ --dry-run=client -o yaml | $(KC) apply -f -
	@$(KC) apply -f k3s/grafana/manifest.yaml
	@$(KC) rollout restart deployment/grafana

k3s-postgres-up:
	@echo "Deploying PostgreSQL..."
	@$(KC) apply -f k3s/postgres/manifest.yaml
	@$(KC) rollout restart statefulset/postgres-postgresql

# Observability
k3s-status:
	@echo "Namespace $(NS) Overview:"
	@$(KC) get all

# Tail logs for a specific pod name (e.g., make k3s-logs-loki-0 or make k3s-logs-alloy-xyz)
k3s-logs-%:
	@$(KC) logs -f $*

# Backup Strategy (Scales down, archives, scales up)
# This dynamically detects if the resource is a statefulset or deployment.
k3s-backup-%:
	@RESOURCE=$$( $(KC) get statefulset,deployment -o name | grep "/$*" | head -n 1 ); \
	if [ -z "$$RESOURCE" ]; then echo "Error: Resource $* not found in namespace $(NS)"; exit 1; fi; \
	echo "Backing up $$RESOURCE..."; \
	echo "Scaling down..."; \
	$(KC) scale --replicas=0 $$RESOURCE; \
	echo "Waiting for termination..."; \
	sleep 5; \
	echo "Finding volume path..."; \
	# Note: Implementation logic for finding PVC path and archiving to be added based on storage strategy. \
	echo "Archiving data..."; \
	echo "Scaling up..."; \
	$(KC) scale --replicas=1 $$RESOURCE
