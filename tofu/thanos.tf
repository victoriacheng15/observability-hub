resource "helm_release" "thanos" {
  name       = "thanos"
  repository = "oci://registry-1.docker.io/bitnamicharts"
  chart      = "thanos"
  version    = "17.3.1"
  namespace  = kubernetes_namespace.observability.metadata[0].name

  values = [file("${path.module}/../k3s/thanos/values.yaml")]

  depends_on = [kubernetes_namespace.observability]
}
