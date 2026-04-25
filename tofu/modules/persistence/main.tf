# --- Shared Standards ---

locals {
  standards = yamldecode(file("${path.module}/../../../k3s/_standards.yaml")).homelab
}

# --- MQTT Broker (EMQX) ---

resource "helm_release" "emqx" {
  name       = "emqx"
  repository = "https://repos.emqx.io/charts"
  chart      = "emqx"
  version    = var.emqx_chart_version
  namespace  = var.observability_namespace

  values = [
    yamlencode({
      # Single-node optimizations
      replicaCount = 1

      emqxConfig = {
        "listeners.tcp.default.bind" = "0.0.0.0:1883"
      }

      # Standard Resource Limits & Standards
      resources            = local.standards.resources.medium
      revisionHistoryLimit = local.standards.deployment.revision_history_limit

      service = {
        type = "ClusterIP"
      }
    })
  ]
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

# --- CloudNativePG Cluster (Data Plane) ---

resource "kubernetes_manifest" "postgres_cluster" {
  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name      = "postgres-hub"
      namespace = var.databases_namespace
    }
    spec = {
      instances       = 3
      imageName       = var.postgres_config.image
      imagePullPolicy = "IfNotPresent"

      resources = local.standards.resources.standard

      bootstrap = {
        initdb = {
          database = var.postgres_config.database
          owner    = var.postgres_config.owner
          secret = {
            name = "postgres-secret"
          }
        }
      }

      inheritedMetadata = {
        labels = {
          "app.kubernetes.io/feature" = "database-core"
        }
      }

      postgresql = {
        shared_preload_libraries = ["timescaledb", "pg_stat_statements"]
        parameters = {
          "archive_mode"               = "on"
          "archive_timeout"            = "30min"
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

      backup = {
        barmanObjectStore = {
          destinationPath = "https://${var.azure_storage_account_name}.blob.core.windows.net/pg-backup/"
          azureCredentials = {
            connectionString = {
              name = "azure-creds"
              key  = "AZURE_CONNECTION_STRING"
            }
          }
        }
      }

      storage = {
        size         = var.postgres_config.storage_size
        storageClass = var.local_path_storage_class_name
      }

      monitoring = {
        enablePodMonitor = true
      }
    }
  }

  depends_on = [helm_release.cnpg_operator]
}

# --- Postgres: Automated Backup Schedule ---

resource "kubernetes_manifest" "postgres_backup_schedule" {
  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "ScheduledBackup"
    metadata = {
      name      = "postgres-daily-backup"
      namespace = var.databases_namespace
    }
    spec = {
      schedule             = var.postgres_config.backup_schedule
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
    namespace = var.databases_namespace
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
