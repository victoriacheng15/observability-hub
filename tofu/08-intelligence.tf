
resource "kubernetes_persistent_volume_claim_v1" "ollama_data" {
  metadata {
    name      = "ollama-data"
    namespace = kubernetes_namespace_v1.hub.metadata[0].name
  }
  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = local.standards.persistence.storage_class
    resources {
      requests = {
        storage = var.ollama_storage_size
      }
    }
  }
  wait_until_bound = false
}

resource "kubernetes_deployment_v1" "ollama" {
  metadata {
    name      = "ollama"
    namespace = kubernetes_namespace_v1.hub.metadata[0].name
    labels = {
      app                         = "ollama"
      "app.kubernetes.io/feature" = "ai-brain"
    }
  }

  spec {
    replicas               = 1
    revision_history_limit = local.standards.deployment.revision_history_limit

    selector {
      match_labels = {
        app = "ollama"
      }
    }

    template {
      metadata {
        labels = {
          app = "ollama"
        }
      }

      spec {
        container {
          name              = "ollama"
          image             = "ollama/ollama:rocm" # ROCm optimized image
          image_pull_policy = local.standards.deployment.image_pull_policy

          port {
            name           = "http"
            container_port = 11434
          }

          env {
            name  = "OLLAMA_HOST"
            value = "0.0.0.0"
          }

          # 780M (gfx1103) Spoofing for ROCm Support
          env {
            name  = "HSA_OVERRIDE_GFX_VERSION"
            value = "11.0.2"
          }

          resources {
            requests = {
              cpu    = "2000m"
              memory = "4Gi"
            }
            limits = {
              cpu          = "4000m"
              memory       = "8Gi"
              "amd.com/gpu" = 1 # Request the iGPU
            }
          }

          security_context {
            read_only_root_filesystem  = false
            allow_privilege_escalation = local.standards.security.container.allow_privilege_escalation
            capabilities {
              drop = local.standards.security.container.capabilities_drop
            }
          }

          volume_mount {
            name       = "ollama-data"
            mount_path = "/root/.ollama"
          }
        }

        volume {
          name = "ollama-data"
          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim_v1.ollama_data.metadata[0].name
          }
        }
      }
    }
  }
}

resource "kubernetes_service_v1" "ollama" {
  metadata {
    name      = "ollama"
    namespace = kubernetes_namespace_v1.hub.metadata[0].name
  }
  spec {
    selector = {
      app = "ollama"
    }
    port {
      name        = "http"
      port        = 11434
      target_port = 11434
      node_port   = 31434
    }
    type = "NodePort"
  }
}
