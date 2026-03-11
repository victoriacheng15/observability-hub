resource "kubernetes_namespace_v1" "observability" {
  metadata {
    name = var.observability_namespace
  }
}

moved {
  from = kubernetes_namespace.observability
  to   = kubernetes_namespace_v1.observability
}
