# K3s Orchestration
.PHONY: k3s-alloy-up k3s-loki-up k3s-minio-up k3s-tempo-up k3s-otel-up k3s-prometheus-up k3s-status k3s-df k3s-prune k3s-logs-% k3s-backup-% kube-lint

# Maintenance
kube-lint:
	@echo "Linting Kubernetes manifests..."
	$(NIX_RUN) "kube-linter lint k3s/"

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
	@echo "Regenerating Alloy manifest..."
	$(NIX_RUN) "helm template alloy grafana/alloy -f k3s/alloy/values.yaml --namespace $(NS) > k3s/alloy/manifest.yaml"
	@echo "Deploying Alloy..."
	@$(KC) apply -f k3s/alloy/manifest.yaml
	@$(KC) rollout restart daemonset/alloy

k3s-loki-up:
	@echo "Regenerating Loki manifest..."
	$(NIX_RUN) "helm template loki grafana/loki -f k3s/loki/values.yaml --namespace $(NS) > k3s/loki/manifest.yaml"
	@echo "Deploying Loki..."
	@$(KC) apply -f k3s/loki/manifest.yaml
	@$(KC) rollout restart statefulset/loki

k3s-minio-up:
	@echo "Deploying MinIO..."
	@$(KC) apply -f k3s/minio/manifest.yaml
	@$(KC) rollout restart deployment/minio

k3s-tempo-up:
	@echo "Regenerating Tempo manifest..."
	$(NIX_RUN) "helm template tempo grafana-community/tempo -f k3s/tempo/values.yaml --namespace $(NS) > k3s/tempo/manifest.yaml"
	@echo "Deploying Tempo..."
	@$(KC) apply -f k3s/tempo/manifest.yaml
	@$(KC) rollout restart statefulset/tempo

k3s-otel-up:
	@echo "Regenerating OTel Collector manifest..."
	$(NIX_RUN) "helm template opentelemetry open-telemetry/opentelemetry-collector -f k3s/opentelemetry/values.yaml --namespace $(NS) > k3s/opentelemetry/manifest.yaml"
	@echo "Deploying OTel Collector..."
	@$(KC) apply -f k3s/opentelemetry/manifest.yaml
	@$(KC) rollout restart deployment/opentelemetry

k3s-prometheus-up:
	@echo "Regenerating Prometheus manifest..."
	$(NIX_RUN) "helm template prometheus prometheus-community/prometheus -f k3s/prometheus/values.yaml --namespace $(NS) > k3s/prometheus/manifest.yaml"
	@echo "Deploying Prometheus..."
	@$(KC) apply -f k3s/prometheus/manifest.yaml
	@$(KC) rollout restart deployment/prometheus-server

k3s-grafana-up:
	@echo "Regenerating Grafana manifest..."
	$(NIX_RUN) "helm template grafana grafana-community/grafana -f k3s/grafana/values.yaml --namespace $(NS) > k3s/grafana/manifest.yaml"
	@echo "Deploying Grafana..."
	@$(KC) create configmap grafana-dashboards --namespace $(NS) --from-file=k3s/grafana/dashboards/ --dry-run=client -o yaml | $(KC) apply -f -
	@$(KC) apply -f k3s/grafana/manifest.yaml
	@$(KC) rollout restart deployment/grafana

k3s-postgres-up:
	@echo "Regenerating PostgreSQL manifest..."
	$(NIX_RUN) "helm template postgres bitnami/postgresql -f k3s/postgres/values.yaml --namespace $(NS) > k3s/postgres/manifest.yaml"
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
