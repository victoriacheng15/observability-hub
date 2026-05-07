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

# --- Postgres: Backup Configuration ---

resource "kubernetes_manifest" "postgres_backup_config" {
  manifest = {
    apiVersion = "postgresql.cnpg.io/v1"
    kind       = "Cluster"
    metadata = {
      name      = "postgres-hub"
      namespace = var.databases_namespace
    }
    spec = {
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
    }
  }

  field_manager {
    name            = "opentofu-cnpg-backup"
    force_conflicts = true
  }
}
