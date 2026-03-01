# K3s Orchestration
.PHONY: k3s-collectors-up k3s-status k3s-df k3s-prune k3s-logs-% k3s-backup-% kube-lint

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

k3s-collectors-up:
	@echo "Regenerating Collectors manifest..."
	$(NIX_RUN) "helm template collectors k3s/collectors -f k3s/collectors/values.yaml --namespace $(NS) > k3s/collectors/manifest.yaml"
	@echo "Deploying Collectors..."
	@$(KC) apply -f k3s/collectors/manifest.yaml
	@$(KC) rollout restart daemonset/collectors

# Observability
k3s-status:
	@echo "Namespace $(NS) Overview:"
	@$(KC) get all

# Tail logs for a specific pod name (e.g., make k3s-logs-loki-0 or make k3s-logs-alloy-xyz)
k3s-logs-%:
	@$(KC) logs -f $*

# Backup Strategy (Scales down, archives, scales up)
# This dynamically detects if the resource is a statefulset or deployment and finds its local-path PVC.
k3s-backup-%:
	@RESOURCE=$$( $(KC) get statefulset,deployment -o name | grep "/$*" | head -n 1 ); \
	if [ -z "$$RESOURCE" ]; then echo "Error: Resource $* not found in namespace $(NS)"; exit 1; fi; \
	PVC_NAME=$$( $(KC) get $$RESOURCE -o jsonpath='{.spec.template.spec.volumes[?(@.persistentVolumeClaim)].persistentVolumeClaim.claimName}' ); \
	if [ -z "$$PVC_NAME" ]; then \
		PVC_NAME=$$( $(KC) get $$RESOURCE -o jsonpath='{.spec.volumeClaimTemplates[0].metadata.name}' ); \
	fi; \
	if [ -z "$$PVC_NAME" ]; then echo "Error: No PVC found for $$RESOURCE"; exit 1; fi; \
	VOLUME_NAME=$$( $(KC) get pvc $$PVC_NAME -n $(NS) -o jsonpath='{.spec.volumeName}' ); \
	echo "Backing up $$RESOURCE (PVC: $$PVC_NAME, Volume: $$VOLUME_NAME)..."; \
	echo "Scaling down..."; \
	$(KC) scale --replicas=0 $$RESOURCE; \
	echo "Waiting for pods to terminate..."; \
	$(KC) wait --for=delete pod -l $$( $(KC) get $$RESOURCE -o jsonpath='{.spec.selector.matchLabels}' | jq -r 'to_entries | .[] | .key + "=" + .value' | paste -sd "," - ) --timeout=60s || true; \
	BACKUP_DIR="/home/server/backups/manual"; \
	sudo mkdir -p $$BACKUP_DIR; \
	TIMESTAMP=$$(date +%Y%m%d_%H%M%S); \
	BACKUP_PATH="$$BACKUP_DIR/$*_"$$TIMESTAMP".tar.gz"; \
	echo "Archiving data from /var/lib/rancher/k3s/storage/ to $$BACKUP_PATH..."; \
	sudo tar -czf $$BACKUP_PATH -C /var/lib/rancher/k3s/storage/ $$(sudo ls /var/lib/rancher/k3s/storage/ | grep "$$VOLUME_NAME"); \
	echo "Scaling up..."; \
	$(KC) scale --replicas=1 $$RESOURCE; \
	echo "Backup complete: $$BACKUP_PATH"
