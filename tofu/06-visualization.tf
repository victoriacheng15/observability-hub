# --- Visualization (Grafana) ---

resource "helm_release" "grafana" {
  name       = "grafana"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "grafana"
  version    = var.grafana_chart_version
  namespace  = kubernetes_namespace_v1.hub.metadata[0].name

  values = [
    file("${path.module}/../k3s/grafana/values.yaml"),
    yamlencode({
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
      persistence = {
        storageClass = local.standards.persistence.storage_class
        size         = local.standards.persistence.size
      }
      podSecurityContext = {
        runAsNonRoot = false
        runAsUser    = 0
        runAsGroup   = 0
        fsGroup      = 472
      }
      containerSecurityContext = {
        readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
        allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
        runAsNonRoot             = true
        runAsUser                = 472
        runAsGroup               = 472
        capabilities = {
          drop = local.standards.security.container.capabilities_drop
        }
      }
      initChownData = {
        securityContext = {
          readOnlyRootFilesystem   = local.standards.exceptions.grafana.init_chown.container_read_only_root_fs
          allowPrivilegeEscalation = local.standards.exceptions.grafana.init_chown.allow_privilege_escalation
          runAsNonRoot             = false
          runAsUser                = 0
          capabilities = {
            add  = local.standards.exceptions.grafana.init_chown.add_capabilities
            drop = local.standards.security.container.capabilities_drop
          }
        }
      }
      resources = local.standards.resources.medium
    })
  ]

  depends_on = [kubernetes_namespace_v1.hub]
}

resource "grafana_folder" "observability" {
  title = "Observability"
}

resource "grafana_dashboard" "dashboards" {
  for_each = fileset("${path.module}/../k3s/grafana/dashboards", "*.json")

  folder      = grafana_folder.observability.id
  config_json = file("${path.module}/../k3s/grafana/dashboards/${each.value}")
  overwrite   = true
}

# --- Analytics (Resource-to-Value Engine) ---
