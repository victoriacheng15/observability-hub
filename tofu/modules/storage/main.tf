terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 3.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = ">= 3.1"
    }
  }
}

# --- Shared Standards ---

locals {
  standards = yamldecode(file("${path.module}/../../../k3s/_standards.yaml")).homelab
}

# --- Azure Storage ---

data "azurerm_storage_account" "hub" {
  name                = var.azurerm_storage_account_name
  resource_group_name = var.azurerm_resource_group_name
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

# --- Object Storage (MinIO) ---

resource "helm_release" "minio" {
  name       = "minio"
  repository = "https://charts.min.io/"
  chart      = "minio"
  version    = var.minio_chart_version
  namespace  = var.databases_namespace

  values = [
    file("${path.module}/../../../k3s/base/infra/minio/values.yaml"),
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
}
