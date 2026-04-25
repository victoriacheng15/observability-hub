variable "azurerm_storage_account_name" {
  description = "Name of the Azure Storage Account."
  type        = string
}

variable "azurerm_resource_group_name" {
  description = "Name of the Azure Resource Group."
  type        = string
}

variable "minio_chart_version" {
  description = "Helm chart version for MinIO."
  type        = string
}

variable "databases_namespace" {
  description = "Namespace where MinIO will be deployed."
  type        = string
}
