# --- MQTT Broker (EMQX) ---

resource "helm_release" "emqx" {
  name       = "emqx"
  repository = "https://repos.emqx.io/charts"
  chart      = "emqx"
  version    = var.emqx_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

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

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- L7 MQTT Visibility Policy ---

resource "null_resource" "mqtt_visibility" {
  triggers = {
    manifest_sha1 = sha1(file("${path.module}/../k3s/opentelemetry/mqtt-visibility.yaml"))
  }

  provisioner "local-exec" {
    command = "kubectl apply -f ${path.module}/../k3s/opentelemetry/mqtt-visibility.yaml"
  }

  depends_on = [helm_release.emqx]
}
