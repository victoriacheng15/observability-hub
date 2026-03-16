# --- Logs (Loki) ---

resource "helm_release" "loki" {
  name       = "loki"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki"
  version    = var.loki_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    file("${path.module}/../k3s/loki/values.yaml"),
    yamlencode({
      loki = {
        persistence = {
          storageClass = local.standards.persistence.storage_class
          size         = local.standards.persistence.size
        }
      }
      singleBinary = {
        persistence = {
          storageClass = local.standards.persistence.storage_class
          size         = local.standards.persistence.size
        }
        resources = local.standards.resources.standard
        podSecurityContext = {
          runAsNonRoot = local.standards.security.pod.run_as_non_root
          fsGroup      = local.standards.security.pod.fs_group
          runAsUser    = local.standards.security.pod.run_as_user
          runAsGroup   = local.standards.security.pod.run_as_group
        }
        containerSecurityContext = {
          readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
          allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
          capabilities = {
            drop = local.standards.security.container.capabilities_drop
          }
        }
      }
      gateway = {
        deploymentStrategy = {
          type = "Recreate"
        }
        affinity  = null
        resources = local.standards.resources.medium
        containerSecurityContext = {
          readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
          allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
          capabilities = {
            drop = local.standards.security.container.capabilities_drop
          }
        }
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Traces (Tempo) ---

resource "helm_release" "tempo" {
  name       = "tempo"
  repository = "https://grafana-community.github.io/helm-charts"
  chart      = "tempo"
  version    = var.tempo_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    file("${path.module}/../k3s/tempo/values.yaml"),
    yamlencode({
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
      persistence = {
        storageClassName = local.standards.persistence.storage_class
        size             = local.standards.persistence.size
      }
      tempo = {
        resources = local.standards.resources.standard
        securityContext = {
          readOnlyRootFilesystem = local.standards.security.container.read_only_root_fs
        }
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Signal Processing (OpenTelemetry) ---

resource "helm_release" "opentelemetry_collector" {
  name       = "opentelemetry-collector"
  repository = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart      = "opentelemetry-collector"
  version    = var.otel_collector_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    file("${path.module}/../k3s/opentelemetry/values.yaml"),
    yamlencode({
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
      resources            = local.standards.resources.medium
      podSecurityContext = {
        runAsNonRoot = local.standards.security.pod.run_as_non_root
        fsGroup      = local.standards.security.pod.fs_group
        runAsUser    = local.standards.security.pod.run_as_user
        runAsGroup   = local.standards.security.pod.run_as_group
      }
      securityContext = {
        readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
        allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
        capabilities = {
          drop = local.standards.security.container.capabilities_drop
        }
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}
