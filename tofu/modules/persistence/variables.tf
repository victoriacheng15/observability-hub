variable "emqx_chart_version" {
  description = "Helm chart version for EMQX."
  type        = string
}

variable "observability_namespace" {
  description = "Namespace for EMQX broker."
  type        = string
}

variable "databases_namespace" {
  description = "Namespace for database services."
  type        = string
}

variable "azure_storage_account_name" {
  description = "Azure storage account name for PostgreSQL backups."
  type        = string
}
