resource "helm_release" "postgres" {
  name       = "postgres"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "postgresql"
  version    = "18.3.0"
  namespace  = kubernetes_namespace.observability.metadata[0].name

  values = [file("${path.module}/../k3s/postgres/values.yaml")]

  depends_on = [kubernetes_namespace.observability]

  lifecycle {
    ignore_changes = [values]
  }
}
