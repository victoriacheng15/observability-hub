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

resource "kubernetes_storage_class_v1" "local_path_retain" {
  metadata {
    name = "local-path-retain"
  }

  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Retain"
  volume_binding_mode = "WaitForFirstConsumer"
}

# --- Database (Postgres) ---

resource "helm_release" "postgres" {
  name       = "postgres"
  repository = "oci://registry-1.docker.io/bitnamicharts"
  chart      = "postgresql"
  version    = "18.3.0"
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [file("${path.module}/../k3s/postgres/values.yaml")]

  depends_on = [kubernetes_namespace_v1.observability]
}

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
