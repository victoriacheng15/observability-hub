# --- Azure Storage ---

data "azurerm_storage_account" "hub" {
  name                = var.azurerm_storage_account_name
  resource_group_name = var.azurerm_resource_group_name
}

data "azurerm_storage_container" "terraform_state" {
  name               = "terraform-state"
  storage_account_id = data.azurerm_storage_account.hub.id
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

# --- Kubernetes Storage ---

resource "kubernetes_storage_class_v1" "local_path_retain" {
  metadata {
    name = "local-path-retain"
  }

  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Retain"
  volume_binding_mode = "WaitForFirstConsumer"
}
