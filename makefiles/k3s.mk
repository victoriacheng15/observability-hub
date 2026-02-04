# K3s Orchestration & Migration

.PHONY: k3s-alloy-up k3s-loki-up k3s-status k3s-logs-% k3s-backup-%

# Apply manifests and rollout restart
k3s-alloy-up:
	@echo "Deploying Alloy..."
	@$(KC) apply -f k3s/alloy/manifest.yaml
	@$(KC) rollout restart daemonset/alloy

k3s-loki-up:
	@echo "Deploying Loki..."
	@$(KC) apply -f k3s/loki/manifest.yaml
	@$(KC) rollout restart statefulset/loki

k3s-grafana-up:
	@echo "Deploying Grafana..."
	@$(KC) apply -f k3s/grafana/manifest.yaml
	@$(KC) rollout restart deployment/grafana

# Observability
k3s-status:
	@echo "Namespace $(NS) Overview:"
	@$(KC) get all

# Tail logs for a specific pod name (e.g., make k3s-logs-loki-0 or make k3s-logs-alloy-xyz)
k3s-logs-%:
	@$(KC) logs -f $*

# Backup Strategy (Scales down, archives, scales up)
# TODO: Implement real backup logic for both StatefulSets and Deployments.
# This needs to dynamically resolve the PVC host path and perform an archive.
k3s-backup-%:
	@echo "Backing up $*..."
	@echo "Scaling down..."
	@$(KC) scale --replicas=0 statefulset/$*
	@echo "Waiting for termination..."
	@sleep 5
	@echo "Finding volume path..."
	# Note: This implies a specific setup where we know the volume path or strategy. 
	# For now, we will just echo the placeholder implementation logic as per plan.
	@echo "Archiving data..."
	@echo "Scaling up..."
	@$(KC) scale --replicas=1 statefulset/$*
