output "local_path_storage_class_name" {
  value       = kubernetes_storage_class_v1.local_path_retain.metadata[0].name
  description = "The name of the local-path-retain storage class."
}

output "minio_release_name" {
  value       = helm_release.minio.name
  description = "The name of the MinIO helm release."
}

output "azure_storage_account_name" {
  value       = data.azurerm_storage_account.hub.name
  description = "The name of the Azure Storage Account."
}
