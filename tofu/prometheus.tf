resource "helm_release" "prometheus" {
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "prometheus"
  namespace  = kubernetes_namespace.observability.metadata[0].name

  values = [file("${path.module}/../k3s/prometheus/values.yaml")]

  depends_on = [kubernetes_namespace.observability]
}
