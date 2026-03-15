# --- Object Storage (MinIO) ---

resource "helm_release" "minio" {
  name       = "minio"
  repository = "https://charts.min.io/"
  chart      = "minio"
  version    = var.minio_chart_version
  namespace  = kubernetes_namespace_v1.databases.metadata[0].name

  values = [
    file("${path.module}/../k3s/minio/values.yaml"),
    yamlencode({
      persistence = {
        storageClass = local.standards.persistence.storage_class
        size         = local.standards.persistence.size
      }
      resources = local.standards.resources.large
      securityContext = {
        enabled      = true
        runAsNonRoot = local.standards.security.pod.run_as_non_root
        fsGroup      = local.standards.security.pod.fs_group
        runAsUser    = local.standards.security.pod.run_as_user
        runAsGroup   = local.standards.security.pod.run_as_group
      }
      containerSecurityContext = {
        readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
        allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
        capabilities = {
          drop = local.standards.security.container.capabilities_drop
        }
      }
      postJob = {
        securityContext = {
          enabled      = true
          runAsNonRoot = local.standards.security.pod.run_as_non_root
          fsGroup      = local.standards.security.pod.fs_group
          runAsUser    = local.standards.security.pod.run_as_user
          runAsGroup   = local.standards.security.pod.run_as_group
        }
      }
      makeBucketJob = {
        securityContext = {
          enabled      = true
          runAsNonRoot = local.standards.security.pod.run_as_non_root
          runAsUser    = local.standards.security.pod.run_as_user
        }
        containerSecurityContext = {
          readOnlyRootFilesystem = local.standards.exceptions.minio.make_bucket_job_read_only_root_fs
        }
      }
      makeUserJob = {
        securityContext = {
          enabled      = true
          runAsNonRoot = local.standards.security.pod.run_as_non_root
          runAsUser    = local.standards.security.pod.run_as_user
        }
        containerSecurityContext = {
          readOnlyRootFilesystem = local.standards.exceptions.minio.make_user_job_read_only_root_fs
        }
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.databases]
}

# --- CloudNativePG Operator (Control Plane) ---

resource "helm_release" "cnpg_operator" {
  name             = "cloudnative-pg"
  repository       = "https://cloudnative-pg.github.io/charts"
  chart            = "cloudnative-pg"
  version          = var.cnpg_operator_chart_version
  namespace        = "cnpg-system"
  create_namespace = true
}

# --- Database Management (pgAdmin Native) ---

resource "kubernetes_persistent_volume_claim_v1" "pgadmin" {
  metadata {
    name      = "pgadmin-pgadmin4"
    namespace = kubernetes_namespace_v1.databases.metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "pgadmin4"
    }
  }

  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = "local-path-retain"
    resources {
      requests = {
        storage = "1Gi"
      }
    }
  }

  wait_until_bound = false
}

resource "kubernetes_config_map_v1" "pgadmin_servers" {
  metadata {
    name      = "pgadmin-pgadmin4-server-definitions"
    namespace = kubernetes_namespace_v1.databases.metadata[0].name
  }

  data = {
    "servers.json" = jsonencode({
      Servers = {
        "postgres-hub" = {
          Group         = "Observability Hub"
          Host          = "postgres-hub-rw.databases.svc.cluster.local"
          MaintenanceDB = "homelab"
          Name          = "Postgres Hub"
          Port          = 5432
          SSLMode       = "prefer"
          Username      = "server"
        }
      }
    })
  }
}

resource "kubernetes_service_v1" "pgadmin" {
  metadata {
    name      = "pgadmin-pgadmin4"
    namespace = kubernetes_namespace_v1.databases.metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "pgadmin4"
    }
  }

  spec {
    selector = {
      "app.kubernetes.io/name"     = "pgadmin4"
      "app.kubernetes.io/instance" = "pgadmin"
    }

    port {
      name        = "http"
      port        = 80
      target_port = 80
      node_port   = 30080
    }

    type = "NodePort"
  }
}

resource "kubernetes_deployment_v1" "pgadmin" {
  metadata {
    name      = "pgadmin-pgadmin4"
    namespace = kubernetes_namespace_v1.databases.metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "pgadmin4"
    }
  }

  spec {
    replicas = 1

    selector {
      match_labels = {
        "app.kubernetes.io/name"     = "pgadmin4"
        "app.kubernetes.io/instance" = "pgadmin"
      }
    }

    template {
      metadata {
        labels = {
          "app.kubernetes.io/name"     = "pgadmin4"
          "app.kubernetes.io/instance" = "pgadmin"
          "app.kubernetes.io/feature"  = "database-management"
        }
      }

      spec {
        automount_service_account_token = false

        container {
          name              = "pgadmin4"
          image             = "docker.io/dpage/pgadmin4:9.11"
          image_pull_policy = "IfNotPresent"

          port {
            name           = "http"
            container_port = 80
          }

          env {
            name  = "PGADMIN_CONFIG_ENHANCED_COOKIE_PROTECTION"
            value = "False"
          }
          env {
            name  = "PGADMIN_DEFAULT_EMAIL"
            value = "admin@observability-hub.home"
          }
          env {
            name = "PGADMIN_DEFAULT_PASSWORD"
            value_from {
              secret_key_ref {
                name = "pgadmin-secret"
                key  = "password"
              }
            }
          }
          env {
            name  = "PGADMIN_SERVER_JSON_FILE"
            value = "/pgadmin4/servers.json"
          }

          volume_mount {
            name       = "pgadmin-data"
            mount_path = "/var/lib/pgadmin"
          }
          volume_mount {
            name       = "definitions"
            mount_path = "/pgadmin4/servers.json"
            sub_path   = "servers.json"
          }

          resources {
            requests = {
              cpu    = "100m"
              memory = "256Mi"
            }
            limits = {
              cpu    = "500m"
              memory = "512Mi"
            }
          }

          liveness_probe {
            http_get {
              path = "/misc/ping"
              port = "http"
            }
            failure_threshold = 3
            period_seconds    = 20
            timeout_seconds   = 5
          }
        }

        volume {
          name = "pgadmin-data"
          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim_v1.pgadmin.metadata[0].name
          }
        }
        volume {
          name = "definitions"
          config_map {
            name = kubernetes_config_map_v1.pgadmin_servers.metadata[0].name
            items {
              key  = "servers.json"
              path = "servers.json"
            }
          }
        }

        security_context {
          fs_group     = 5050
          run_as_group = 5050
          run_as_user  = 5050
        }
      }
    }
  }

  depends_on = [kubernetes_namespace_v1.databases, kubernetes_persistent_volume_claim_v1.pgadmin]
}

# --- CloudNativePG Cluster (Data Plane) ---

resource "kubernetes_manifest" "postgres_cluster" {
  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name      = "postgres-hub"
      namespace = kubernetes_namespace_v1.databases.metadata[0].name
    }
    spec = {
      instances       = 3
      imageName       = var.postgres_image
      imagePullPolicy = "IfNotPresent"

      # Permanent database and user identity
      bootstrap = {
        initdb = {
          database = var.postgres_database
          owner    = var.postgres_owner
          secret = {
            name = "postgres-secret"
          }
        }
      }

      # Correct schema for label propagation to pods

      inheritedMetadata = {
        labels = {
          "app.kubernetes.io/feature" = "database-core"
        }
      }


      # Standard 2026 PostgreSQL parameters
      postgresql = {
        shared_preload_libraries = ["timescaledb", "pg_stat_statements"]
        parameters = {
          "archive_mode"               = "on"
          "archive_timeout"            = "5min"
          "dynamic_shared_memory_type" = "posix"
          "full_page_writes"           = "on"
          "log_destination"            = "csvlog"
          "log_directory"              = "/controller/log"
          "log_filename"               = "postgres"
          "log_rotation_age"           = "0"
          "log_rotation_size"          = "0"
          "log_truncate_on_rotation"   = "false"
          "logging_collector"          = "on"
          "max_parallel_workers"       = "32"
          "max_replication_slots"      = "32"
          "max_worker_processes"       = "32"
          "shared_memory_type"         = "mmap"
          "shared_preload_libraries"   = ""
          "ssl_max_protocol_version"   = "TLSv1.3"
          "ssl_min_protocol_version"   = "TLSv1.3"
          "wal_keep_size"              = "512MB"
          "wal_level"                  = "logical"
          "wal_log_hints"              = "on"
          "wal_receiver_timeout"       = "5s"
          "wal_sender_timeout"         = "5s"
        }
      }

      # Enterprise Backup (Azure streaming)
      backup = {
        barmanObjectStore = {
          destinationPath = "https://${data.azurerm_storage_account.hub.name}.blob.core.windows.net/pg-backup/"
          azureCredentials = {
            connectionString = {
              name = "azure-creds"
              key  = "AZURE_CONNECTION_STRING"
            }
          }
        }
      }

      storage = {
        size         = var.postgres_storage_size
        storageClass = kubernetes_storage_class_v1.local_path_retain.metadata[0].name
      }

      monitoring = {
        enablePodMonitor = true
      }
    }
  }

  depends_on = [helm_release.cnpg_operator, azurerm_storage_container.pg_backup, kubernetes_namespace_v1.databases]
}

# --- Postgres: Automated Backup Schedule ---

resource "kubernetes_manifest" "postgres_backup_schedule" {
  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "ScheduledBackup"
    metadata = {
      name      = "postgres-daily-backup"
      namespace = kubernetes_namespace_v1.databases.metadata[0].name
    }
    spec = {
      schedule             = var.postgres_backup_schedule
      backupOwnerReference = "self"
      cluster = {
        name = "postgres-hub"
      }
    }
  }

  depends_on = [kubernetes_manifest.postgres_cluster]
}

# --- Host Access: Postgres NodePort ---

resource "kubernetes_service_v1" "postgres_nodeport" {
  metadata {
    name      = "postgres-host-access"
    namespace = kubernetes_namespace_v1.databases.metadata[0].name
  }
  spec {
    selector = {
      "cnpg.io/cluster" = "postgres-hub"
      "role"            = "primary"
    }
    port {
      port        = 5432
      target_port = 5432
      node_port   = var.postgres_node_port
    }
    type = "NodePort"
  }
}
