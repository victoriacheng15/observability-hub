resource "helm_release" "grafana" {
  name       = "grafana"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "grafana"
  namespace  = kubernetes_namespace.observability.metadata[0].name

  values = [file("${path.module}/../k3s/grafana/values.yaml")]

  depends_on = [kubernetes_namespace.observability]
}
