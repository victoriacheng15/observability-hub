resource "helm_release" "prometheus" {
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "prometheus"
  version    = "28.10.1"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [file("${path.module}/../k3s/prometheus/values.yaml")]

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

