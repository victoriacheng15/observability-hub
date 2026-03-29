# --- Visualization (Grafana) ---

resource "helm_release" "grafana" {
  name       = "grafana"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "grafana"
  version    = var.grafana_chart_version
  namespace  = kubernetes_namespace_v1.hub.metadata[0].name

  values = [
    file("${path.module}/../k3s/base/infra/grafana/values.yaml"),
    yamlencode({
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
      persistence = {
        storageClass = local.standards.persistence.storage_class
        size         = local.standards.persistence.size
      }
      podSecurityContext = {
        runAsNonRoot = false
        runAsUser    = 0
        runAsGroup   = 0
        fsGroup      = 472
      }
      containerSecurityContext = {
        readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
        allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
        runAsNonRoot             = true
        runAsUser                = 472
        runAsGroup               = 472
        capabilities = {
          drop = local.standards.security.container.capabilities_drop
        }
      }
      initChownData = {
        securityContext = {
          readOnlyRootFilesystem   = local.standards.exceptions.grafana.init_chown.container_read_only_root_fs
          allowPrivilegeEscalation = local.standards.exceptions.grafana.init_chown.allow_privilege_escalation
          runAsNonRoot             = false
          runAsUser                = 0
          capabilities = {
            add  = local.standards.exceptions.grafana.init_chown.add_capabilities
            drop = local.standards.security.container.capabilities_drop
          }
        }
      }
      resources = local.standards.resources.medium
    })
  ]

  depends_on = [kubernetes_namespace_v1.hub]
}

resource "grafana_folder" "observability" {
  title = "Observability"
}

resource "grafana_dashboard" "dashboards" {
  for_each = fileset("${path.module}/../k3s/base/infra/grafana/dashboards", "*.json")

  folder      = grafana_folder.observability.id
  config_json = file("${path.module}/../k3s/base/infra/grafana/dashboards/${each.value}")
  overwrite   = true
}

# --- Alerting (Grafana Unified Alerting) ---

locals {
  grafana_alerting_enabled = var.grafana_discord_webhook_url != null && trimspace(var.grafana_discord_webhook_url) != ""
}

resource "grafana_contact_point" "discord" {
  count = local.grafana_alerting_enabled ? 1 : 0

  name = "Discord"

  discord {
    url     = var.grafana_discord_webhook_url
    title   = "{{ .Status | toUpper }}: {{ .CommonLabels.alertname }}"
    message = <<-EOT
    {{ len .Alerts.Firing }} firing, {{ len .Alerts.Resolved }} resolved
    {{ range .Alerts -}}
    - {{ index .Labels "alertname" }}: {{ index .Annotations "summary" }}
    {{ end -}}
    EOT
  }
}

resource "grafana_notification_policy" "default" {
  count = local.grafana_alerting_enabled ? 1 : 0

  contact_point   = grafana_contact_point.discord[0].name
  group_by        = ["grafana_folder", "alertname"]
  group_wait      = "30s"
  group_interval  = "5m"
  repeat_interval = "4h"
}

resource "grafana_rule_group" "cilium_hubble" {
  count = local.grafana_alerting_enabled ? 1 : 0

  name             = "Cilium Hubble Alerts"
  folder_uid       = grafana_folder.observability.uid
  interval_seconds = 60

  rule {
    name      = "Hubble Metrics Missing"
    condition = "B"
    for       = "10m"

    annotations = {
      summary     = "Prometheus cannot scrape Hubble metrics."
      description = "The Hubble metrics target has stayed below 1 for 10 minutes."
    }

    labels = {
      severity = "warning"
      service  = "hubble"
    }

    exec_err_state = "Alerting"
    no_data_state  = "Alerting"
    is_paused      = false

    data {
      ref_id         = "A"
      query_type     = ""
      datasource_uid = "prometheus-provisioned"

      relative_time_range {
        from = 600
        to   = 0
      }

      model = jsonencode({
        datasource = {
          type = "prometheus"
          uid  = "prometheus-provisioned"
        }
        editorMode    = "code"
        exemplar      = false
        expr          = "max_over_time(up{job=\"hubble\"}[5m])"
        instant       = true
        intervalMs    = 1000
        legendFormat  = "__auto"
        maxDataPoints = 43200
        range         = false
        refId         = "A"
      })
    }

    data {
      ref_id         = "B"
      query_type     = ""
      datasource_uid = "-100"

      relative_time_range {
        from = 0
        to   = 0
      }

      model = jsonencode({
        conditions = [
          {
            evaluator = {
              params = [1]
              type   = "lt"
            }
            operator = {
              type = "and"
            }
            query = {
              params = ["A"]
            }
            reducer = {
              params = []
              type   = "last"
            }
            type = "query"
          }
        ]
        datasource = {
          name = "Expression"
          type = "__expr__"
          uid  = "-100"
        }
        intervalMs    = 1000
        maxDataPoints = 43200
        refId         = "B"
        type          = "classic_conditions"
      })
    }
  }

  rule {
    name      = "Hubble Policy Drops High"
    condition = "B"
    for       = "10m"

    annotations = {
      summary     = "Hubble is reporting repeated dropped flows."
      description = "The five minute average drop rate has stayed above 1 event per second for 10 minutes."
    }

    labels = {
      severity = "warning"
      service  = "hubble"
    }

    exec_err_state = "Alerting"
    no_data_state  = "OK"
    is_paused      = false

    data {
      ref_id         = "A"
      query_type     = ""
      datasource_uid = "prometheus-provisioned"

      relative_time_range {
        from = 600
        to   = 0
      }

      model = jsonencode({
        datasource = {
          type = "prometheus"
          uid  = "prometheus-provisioned"
        }
        editorMode    = "code"
        exemplar      = false
        expr          = "sum(rate(hubble_drop_total[5m]))"
        instant       = true
        intervalMs    = 1000
        legendFormat  = "__auto"
        maxDataPoints = 43200
        range         = false
        refId         = "A"
      })
    }

    data {
      ref_id         = "B"
      query_type     = ""
      datasource_uid = "-100"

      relative_time_range {
        from = 0
        to   = 0
      }

      model = jsonencode({
        conditions = [
          {
            evaluator = {
              params = [1]
              type   = "gt"
            }
            operator = {
              type = "and"
            }
            query = {
              params = ["A"]
            }
            reducer = {
              params = []
              type   = "last"
            }
            type = "query"
          }
        ]
        datasource = {
          name = "Expression"
          type = "__expr__"
          uid  = "-100"
        }
        intervalMs    = 1000
        maxDataPoints = 43200
        refId         = "B"
        type          = "classic_conditions"
      })
    }
  }
}
