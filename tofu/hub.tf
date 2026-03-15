# --- Visualization (Grafana) ---

resource "helm_release" "grafana" {
  name       = "grafana"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "grafana"
  version    = var.grafana_chart_version
  namespace  = kubernetes_namespace_v1.hub.metadata[0].name

  values = [
    file("${path.module}/../k3s/grafana/values.yaml"),
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
  for_each = fileset("${path.module}/../k3s/grafana/dashboards", "*.json")

  folder      = grafana_folder.observability.id
  config_json = file("${path.module}/../k3s/grafana/dashboards/${each.value}")
  overwrite   = true
}

# --- Analytics (Resource-to-Value Engine) ---

resource "kubernetes_daemon_set_v1" "analytics" {
  metadata {
    name      = "analytics"
    namespace = kubernetes_namespace_v1.hub.metadata[0].name
    labels = {
      "app.kubernetes.io/name"    = "analytics"
      "app.kubernetes.io/feature" = "analytics-engine"
    }
  }

  spec {
    selector {
      match_labels = {
        "app.kubernetes.io/name" = "analytics"
      }
    }

    template {
      metadata {
        labels = {
          "app.kubernetes.io/name" = "analytics"
        }
      }

      spec {
        host_network = true
        dns_policy   = "ClusterFirstWithHostNet"

        container {
          name              = "analytics"
          image             = "analytics:v0.1.0"
          image_pull_policy = "IfNotPresent"

          # Observability Endpoints (FQDN)
          env {
            name  = "THANOS_URL"
            value = "http://thanos-query.observability.svc.cluster.local:9090"
          }
          env {
            name  = "OTEL_EXPORTER_OTLP_ENDPOINT"
            value = "opentelemetry.observability.svc.cluster.local:4317"
          }

          # Database Credentials
          env {
            name  = "DB_HOST"
            value = "postgres-hub-rw.databases.svc.cluster.local"
          }
          env {
            name  = "DB_PORT"
            value = "5432"
          }
          env {
            name  = "DB_USER"
            value = "server"
          }
          env {
            name  = "DB_NAME"
            value = "homelab"
          }
          env {
            name = "SERVER_DB_PASSWORD"
            value_from {
              secret_key_ref {
                name = "postgres-secret"
                key  = "server-db-password"
              }
            }
          }

          resources {
            requests = {
              cpu    = "100m"
              memory = "200Mi"
            }
            limits = {
              cpu    = "100m"
              memory = "400Mi"
            }
          }

          volume_mount {
            name       = "tailscale-sock"
            mount_path = "/var/run/tailscale/tailscaled.sock"
            read_only  = true
          }
          volume_mount {
            name       = "host-hostname"
            mount_path = "/etc/host_hostname"
            read_only  = true
          }
          volume_mount {
            name       = "host-os-release"
            mount_path = "/etc/host_os-release"
            read_only  = true
          }
        }

        volume {
          name = "tailscale-sock"
          host_path {
            path = "/var/run/tailscale/tailscaled.sock"
            type = "Socket"
          }
        }
        volume {
          name = "host-hostname"
          host_path {
            path = "/etc/hostname"
            type = "File"
          }
        }
        volume {
          name = "host-os-release"
          host_path {
            path = "/etc/os-release"
            type = "File"
          }
        }
      }
    }
  }

  depends_on = [kubernetes_namespace_v1.hub]
}
