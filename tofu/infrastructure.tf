# --- Storage ---

data "azurerm_storage_account" "hub" {
  name                = "observabilityhub"
  resource_group_name = "personal-rg"
}

resource "azurerm_storage_container" "terraform_state" {
  name                  = "terraform-state"
  storage_account_id    = data.azurerm_storage_account.hub.id
  container_access_type = "private"

  # Prevent accidental deletion via Tofu
  lifecycle {
    prevent_destroy = true
  }
}

resource "azurerm_storage_container" "pg_backup" {
  name                  = "pg-backup"
  storage_account_id    = data.azurerm_storage_account.hub.id
  container_access_type = "private"
}

# --- FinOps: Storage Lifecycle ---

resource "azurerm_storage_management_policy" "pg_backup_lifecycle" {
  storage_account_id = data.azurerm_storage_account.hub.id

  rule {
    name    = "ArchiveOldBackups"
    enabled = true
    filters {
      prefix_match = ["pg-backup/"]
      blob_types   = ["blockBlob"]
    }
    actions {
      base_blob {
        tier_to_archive_after_days_since_modification_greater_than = 90
        delete_after_days_since_modification_greater_than          = 365
      }
    }
  }
}

resource "kubernetes_storage_class_v1" "local_path_retain" {
  metadata {
    name = "local-path-retain"
  }

  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Retain"
  volume_binding_mode = "WaitForFirstConsumer"
}

# --- Database (Postgres) ---

# --- Object Storage (MinIO) ---

resource "helm_release" "minio" {
  name       = "minio"
  repository = "https://charts.min.io/"
  chart      = "minio"
  version    = "5.4.0"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [file("${path.module}/../k3s/minio/values.yaml")]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- CloudNativePG Operator (Control Plane) ---

resource "helm_release" "cnpg_operator" {
  name       = "cloudnative-pg"
  repository = "https://cloudnative-pg.github.io/charts"
  chart      = "cloudnative-pg"
  version    = "0.23.0" # Standard 2026 stable release
  namespace  = "cnpg-system"
  create_namespace = true
}

# --- CloudNativePG Cluster (Data Plane) ---

resource "kubernetes_manifest" "postgres_cluster" {
  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name      = "postgres"
      namespace = kubernetes_namespace_v1.observability.metadata[0].name
    }
    spec = {
      instances       = 3
      imageName       = "localhost/postgres-cnpg:17"
      imagePullPolicy = "IfNotPresent"

      # Permanent database and user identity
      bootstrap = {
        initdb = {
          database = "homelab"
          owner    = "server"
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
        size         = "10Gi"
        storageClass = kubernetes_storage_class_v1.local_path_retain.metadata[0].name
      }

      monitoring = {
        enablePodMonitor = true
      }
    }
  }

  depends_on = [helm_release.cnpg_operator, azurerm_storage_container.pg_backup]
}

# --- Postgres: Automated Backup Schedule ---

resource "kubernetes_manifest" "postgres_backup_schedule" {
  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "ScheduledBackup"
    metadata = {
      name      = "postgres-daily-backup"
      namespace = kubernetes_namespace_v1.observability.metadata[0].name
    }
    spec = {
      schedule = "0 0 2 * * *" # Daily at 2 AM
      backupOwnerReference = "self"
      cluster = {
        name = "postgres"
      }
    }
  }

  depends_on = [kubernetes_manifest.postgres_cluster]
}

# --- Host Access: Postgres NodePort ---

resource "kubernetes_service_v1" "postgres_nodeport" {
  metadata {
    name      = "postgres-host-access"
    namespace = kubernetes_namespace_v1.observability.metadata[0].name
  }
  spec {
    selector = {
      "cnpg.io/cluster" = "postgres"
      "role"            = "primary"
    }
    port {
      port        = 5432
      target_port = 5432
      node_port   = 30432
    }
    type = "NodePort"
  }
}
