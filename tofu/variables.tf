# --- Environment ---

variable "kubeconfig_path" {
  description = "Path to the kubeconfig file."
  type        = string
  default     = "~/.kube/config"
}

# --- Namespaces ---

variable "observability_namespace" {
  description = "Namespace for all observability services."
  type        = string
  default     = "observability"
}

variable "databases_namespace" {
  description = "Namespace for all database and persistence services."
  type        = string
  default     = "databases"
}

variable "hub_namespace" {
  description = "Namespace for analytical and hub-facing services."
  type        = string
  default     = "hub"
}

variable "argocd_namespace" {
  description = "Namespace for ArgoCD GitOps controller."
  type        = string
  default     = "argocd"
}

variable "argocd_chart_version" {
  description = "Helm chart version for ArgoCD."
  type        = string
  default     = "7.7.12"
}

# --- Azure Storage ---

variable "azurerm_storage_account_name" {
  description = "Name of the Azure Storage Account."
  type        = string
  default     = "observabilityhub"
}

variable "azurerm_resource_group_name" {
  description = "Name of the Azure Resource Group."
  type        = string
  default     = "personal-rg"
}

# --- Databases & Persistence (infrastructure.tf) ---

variable "minio_chart_version" {
  description = "Helm chart version for MinIO."
  type        = string
  default     = "5.4.0"
}

variable "cnpg_operator_chart_version" {
  description = "Helm chart version for CloudNativePG Operator."
  type        = string
  default     = "0.23.0"
}

variable "pgadmin_chart_version" {
  description = "Helm chart version for pgAdmin."
  type        = string
  default     = "1.59.0"
}

variable "postgres_image" {
  description = "PostgreSQL image to use in the cluster."
  type        = string
  default     = "localhost/postgres-cnpg:17"
}

variable "postgres_database" {
  description = "Default database name for the PostgreSQL cluster."
  type        = string
  default     = "homelab"
}

variable "postgres_owner" {
  description = "Default owner for the PostgreSQL database."
  type        = string
  default     = "server"
}

variable "postgres_storage_size" {
  description = "Storage size for the PostgreSQL cluster."
  type        = string
  default     = "10Gi"
}

variable "postgres_backup_schedule" {
  description = "Cron schedule for automated PostgreSQL backups."
  type        = string
  default     = "0 0 2 * * *"
}

variable "postgres_node_port" {
  description = "NodePort for external PostgreSQL access."
  type        = number
  default     = 30432
}

# --- Observability Stack (observability.tf) ---

variable "prometheus_chart_version" {
  description = "Helm chart version for Prometheus."
  type        = string
  default     = "28.10.1"
}

variable "kepler_chart_version" {
  description = "Helm chart version for Kepler."
  type        = string
  default     = "0.11.2"
}

variable "thanos_chart_version" {
  description = "Helm chart version for Thanos."
  type        = string
  default     = "17.3.1"
}

variable "loki_chart_version" {
  description = "Helm chart version for Loki."
  type        = string
  default     = "6.53.0"
}

variable "tempo_chart_version" {
  description = "Helm chart version for Tempo."
  type        = string
  default     = "1.26.1"
}

variable "otel_collector_chart_version" {
  description = "Helm chart version for OpenTelemetry Collector."
  type        = string
  default     = "0.146.0"
}

variable "grafana_chart_version" {
  description = "Helm chart version for Grafana."
  type        = string
  default     = "10.5.15"
}

variable "n8n_chart_version" {
  description = "Helm chart version for n8n."
  type        = string
  default     = "1.16.31"
}

variable "ollama_storage_size" {
  description = "Storage size for Ollama models."
  type        = string
  default     = "50Gi"
}

variable "emqx_chart_version" {
  description = "Helm chart version for EMQX."
  type        = string
  default     = "5.8.9"
}

variable "cilium_chart_version" {
  description = "Helm chart version for Cilium."
  type        = string
  default     = "1.16.1"
}

variable "amdgpu_plugin_chart_version" {
  description = "Helm chart version for AMD GPU Device Plugin."
  type        = string
  default     = "0.21.0"
}
