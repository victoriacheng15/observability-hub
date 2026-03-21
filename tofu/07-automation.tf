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
            requests = local.standards.resources.small.requests
            limits   = local.standards.resources.small.limits
          }

          volume_mount {
            name       = "tailscale-sock"
            mount_path = "/var/run/tailscale"
            read_only  = false
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
            path = "/run/tailscale"
            type = "Directory"
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

# --- Workflow Orchestration (n8n Native) ---

resource "kubernetes_persistent_volume_claim_v1" "n8n_data" {
  metadata {
    name      = "n8n-data"
    namespace = kubernetes_namespace_v1.hub.metadata[0].name
  }
  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = local.standards.persistence.storage_class
    resources {
      requests = {
        storage = local.standards.persistence.size
      }
    }
  }
  wait_until_bound = false
}

resource "kubernetes_deployment_v1" "n8n" {
  metadata {
    name      = "n8n"
    namespace = kubernetes_namespace_v1.hub.metadata[0].name
    labels = {
      app                         = "n8n"
      "app.kubernetes.io/feature" = "automation"
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        app = "n8n"
      }
    }

    template {
      metadata {
        labels = {
          app = "n8n"
        }
      }

      spec {
        container {
          name  = "n8n"
          image = "n8nio/n8n:latest"

          port {
            name           = "http"
            container_port = 5678
          }

          # Using postgresdb type to match n8n standards
          env {
            name  = "DB_TYPE"
            value = "postgresdb"
          }
          env {
            name  = "DB_POSTGRESDB_HOST"
            value = "postgres-hub-rw.databases.svc.cluster.local"
          }
          env {
            name  = "DB_POSTGRESDB_PORT"
            value = "5432"
          }
          env {
            name  = "DB_POSTGRESDB_DATABASE"
            value = "n8n"
          }
          env {
            name  = "DB_POSTGRESDB_USER"
            value = var.postgres_owner
          }
          env {
            name = "DB_POSTGRESDB_PASSWORD"
            value_from {
              secret_key_ref {
                name = "postgres-secret"
                key  = "server-db-password"
              }
            }
          }
          env {
            name  = "GENERIC_TIMEZONE"
            value = "Asia/Singapore"
          }
          env {
            name  = "N8N_PORT"
            value = "5678"
          }
          env {
            name  = "N8N_SECURE_COOKIE"
            value = "false"
          }
          # Reliability tweaks
          env {
            name  = "N8N_SKIP_WEBHOOK_DEREGISTRATION_ON_SHUTDOWN"
            value = "true"
          }
          env {
            name  = "N8N_DISABLE_VERSION_CHECK"
            value = "true"
          }

          volume_mount {
            name       = "n8n-data"
            mount_path = "/home/node/.n8n"
          }

          liveness_probe {
            http_get {
              path = "/healthz"
              port = "http"
            }
            initial_delay_seconds = 60
            period_seconds        = 30
            timeout_seconds       = 10
            failure_threshold     = 5
          }

          readiness_probe {
            http_get {
              path = "/healthz"
              port = "http"
            }
            initial_delay_seconds = 60
            period_seconds        = 10
            timeout_seconds       = 5
            failure_threshold     = 3
          }

          resources {
            requests = local.standards.resources.standard.requests
            limits   = local.standards.resources.standard.limits
          }
        }

        volume {
          name = "n8n-data"
          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim_v1.n8n_data.metadata[0].name
          }
        }
      }
    }
  }
}

# Host Access: n8n NodePort
resource "kubernetes_service_v1" "n8n" {
  metadata {
    name      = "n8n"
    namespace = kubernetes_namespace_v1.hub.metadata[0].name
  }
  spec {
    selector = {
      app = "n8n"
    }
    port {
      port        = 80
      target_port = 5678
      node_port   = 30568
    }
    type = "NodePort"
  }
}

