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
