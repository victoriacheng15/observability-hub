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

resource "grafana_rule_group" "loki_log_errors" {
  count = local.grafana_alerting_enabled ? 1 : 0

  name             = "Loki Log Alerts"
  folder_uid       = grafana_folder.observability.uid
  interval_seconds = 600

  rule {
    name      = "Error Logs Detected"
    condition = "B"
    for       = "0s"

    annotations = {
      summary     = "Loki received error-level log output."
      description = "At least one log line containing error, fatal, or panic appeared in a service_name-labeled Loki stream during the last 5 minutes."
    }

    labels = {
      severity = "warning"
      service  = "logs"
    }

    exec_err_state = "Alerting"
    no_data_state  = "OK"
    is_paused      = false

    data {
      ref_id         = "A"
      query_type     = ""
      datasource_uid = "loki-provisioned"

      relative_time_range {
        from = 300
        to   = 0
      }

      model = jsonencode({
        datasource = {
          type = "loki"
          uid  = "loki-provisioned"
        }
        editorMode    = "code"
        expr          = "sum(count_over_time({service_name=~\".+\"} |~ \"(?i)\\b(error|fatal|panic)\\b\" [5m]))"
        intervalMs    = 1000
        maxDataPoints = 43200
        queryType     = "range"
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
              params = [0]
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
