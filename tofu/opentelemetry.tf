resource "helm_release" "opentelemetry_collector" {
  name       = "opentelemetry-collector"
  repository = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart      = "opentelemetry-collector"
  namespace  = kubernetes_namespace.observability.metadata[0].name

  values = [file("${path.module}/../k3s/opentelemetry/values.yaml")]

  depends_on = [kubernetes_namespace.observability]
}
