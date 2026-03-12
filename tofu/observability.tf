# --- Metrics (Prometheus & Kepler) ---

resource "helm_release" "prometheus" {
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "prometheus"
  version    = "28.10.1"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [file("${path.module}/../k3s/prometheus/values.yaml")]

  depends_on = [kubernetes_namespace_v1.observability]
}

resource "helm_release" "kepler" {
  name       = "kepler"
  repository = "oci://quay.io/sustainable_computing_io/charts"
  chart      = "kepler"
  version    = "0.11.2"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [file("${path.module}/../k3s/kepler/values.yaml")]

  depends_on = [kubernetes_namespace_v1.observability]
}

resource "kubernetes_service_v1" "prometheus_thanos_grpc" {
  metadata {
    name      = "prometheus-thanos-grpc"
    namespace = kubernetes_namespace_v1.observability.metadata[0].name
    labels = {
      "app.kubernetes.io/name"      = "prometheus"
      "app.kubernetes.io/component" = "thanos-sidecar"
    }
  }

  spec {
    selector = {
      "app.kubernetes.io/name"      = "prometheus"
      "app.kubernetes.io/component" = "server"
      "app.kubernetes.io/instance"  = "prometheus"
    }

    port {
      name        = "grpc"
      port        = 10901
      target_port = 10901
    }

    type       = "ClusterIP"
    cluster_ip = "None" # Headless service for SRV discovery
  }
}

# --- Long-term Metrics (Thanos) ---

resource "helm_release" "thanos" {
  name       = "thanos"
  repository = "oci://registry-1.docker.io/bitnamicharts"
  chart      = "thanos"
  version    = "17.3.1"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    file("${path.module}/../k3s/thanos/values.yaml"),
    yamlencode({
      query = {
        extraFlags = ["--endpoint=prometheus-thanos-grpc.observability.svc.cluster.local:10901"]
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Logs (Loki) ---

resource "helm_release" "loki" {
  name       = "loki"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki"
  version    = "6.53.0"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [file("${path.module}/../k3s/loki/values.yaml")]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Traces (Tempo) ---

resource "helm_release" "tempo" {
  name       = "tempo"
  repository = "https://grafana-community.github.io/helm-charts"
  chart      = "tempo"
  version    = "1.26.1"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [file("${path.module}/../k3s/tempo/values.yaml")]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Signal Processing (OpenTelemetry) ---

resource "helm_release" "opentelemetry_collector" {
  name       = "opentelemetry-collector"
  repository = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart      = "opentelemetry-collector"
  version    = "0.146.0"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [file("${path.module}/../k3s/opentelemetry/values.yaml")]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Visualization (Grafana) ---

resource "helm_release" "grafana" {
  name       = "grafana"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "grafana"
  version    = "10.5.15"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [file("${path.module}/../k3s/grafana/values.yaml")]

  depends_on = [kubernetes_namespace_v1.observability]
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
