resource "helm_release" "minio" {
  name       = "minio"
  repository = "https://charts.min.io/"
  chart      = "minio"
  namespace  = kubernetes_namespace.observability.metadata[0].name

  values = [file("${path.module}/../k3s/minio/values.yaml")]

  depends_on = [kubernetes_namespace.observability]
}
