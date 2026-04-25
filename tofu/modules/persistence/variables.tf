variable "emqx_chart_version" {
  description = "Helm chart version for EMQX."
  type        = string
}

variable "cnpg_operator_chart_version" {
  description = "Helm chart version for CloudNativePG Operator."
  type        = string
}

variable "postgres_config" {
  description = "PostgreSQL cluster configuration."
  type = object({
    image           = string
    database        = string
    owner           = string
    storage_size    = string
    backup_schedule = string
  })
}

variable "postgres_node_port" {
  description = "NodePort for external PostgreSQL access."
  type        = number
}

variable "databases_namespace" {
  description = "Namespace for database services."
  type        = string
}

variable "observability_namespace" {
  description = "Namespace for EMQX broker."
  type        = string
}

variable "azure_storage_account_name" {
  description = "Azure storage account name for backups."
  type        = string
}

variable "local_path_storage_class_name" {
  description = "Storage class name for local volumes."
  type        = string
}
