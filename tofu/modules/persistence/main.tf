# --- Shared Standards ---

locals {
  standards = yamldecode(file("${path.module}/../../../k3s/_standards.yaml")).homelab
}

# --- MQTT Broker (EMQX) ---

resource "helm_release" "emqx" {
  name       = "emqx"
  repository = "https://repos.emqx.io/charts"
  chart      = "emqx"
  version    = var.emqx_chart_version
  namespace  = var.observability_namespace

  values = [
    yamlencode({
      # Single-node optimizations
      replicaCount = 1

      emqxConfig = {
        "listeners.tcp.default.bind" = "0.0.0.0:1883"
      }

      # Standard Resource Limits & Standards
      resources            = local.standards.resources.medium
      revisionHistoryLimit = local.standards.deployment.revision_history_limit

      service = {
        type = "ClusterIP"
      }
    })
  ]
}
