# --- Global Environment ---

variable "kubeconfig_path" {
  description = "Path to the kubeconfig file."
  type        = string
  default     = "~/.kube/config"
}

variable "tags" {
  description = "Common tags for all project resources."
  type        = map(string)
  default = {
    project       = "observability-hub"
    observability = "enabled"
    managed_by    = "opentofu"
  }
}

# --- Shared Namespaces ---

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

variable "hardware_sim_namespace" {
  description = "Namespace for hardware simulation and chaos experiments."
  type        = string
  default     = "hardware-sim"
}

# --- Module Specific Versions (Kept at root for easier lifecycle management) ---

variable "argocd_chart_version" {
  description = "Helm chart version for ArgoCD."
  type        = string
  default     = "9.5.4"
}

variable "azurerm_storage_account_name" {
  description = "Name of the Azure Storage Account."
  type        = string
  default     = "obshub"
}

variable "azurerm_resource_group_name" {
  description = "Name of the Azure Resource Group."
  type        = string
  default     = "observability-rg"
}

variable "minio_chart_version" {
  description = "Helm chart version for MinIO."
  type        = string
  default     = "5.4.0"
}

variable "cnpg_operator_chart_version" {
  description = "Helm chart version for CloudNativePG Operator."
  type        = string
  default     = "0.27.1"
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
  default = {
    image           = "localhost/postgres-cnpg:17"
    database        = "homelab"
    owner           = "server"
    storage_size    = "10Gi"
    backup_schedule = "0 0 2 * * *"
  }
}

variable "postgres_node_port" {
  description = "NodePort for external PostgreSQL access."
  type        = number
  default     = 30432
}

variable "prometheus_chart_version" {
  description = "Helm chart version for Prometheus."
  type        = string
  default     = "28.14.1"
}

variable "thanos_chart_version" {
  description = "Helm chart version for Thanos."
  type        = string
  default     = "17.3.1"
}

variable "loki_chart_version" {
  description = "Helm chart version for Loki."
  type        = string
  default     = "6.55.0"
}

variable "tempo_chart_version" {
  description = "Helm chart version for Tempo."
  type        = string
  default     = "2.0.0"
}

variable "otel_collector_chart_version" {
  description = "Helm chart version for OpenTelemetry Collector."
  type        = string
  default     = "0.147.1"
}

variable "grafana_chart_version" {
  description = "Helm chart version for Grafana."
  type        = string
  default     = "10.5.15"
}

variable "grafana_discord_webhook_url" {
  description = "Discord webhook URL used by Grafana alert notifications."
  type        = string
  sensitive   = true
  default     = null
}

variable "emqx_chart_version" {
  description = "Helm chart version for EMQX."
  type        = string
  default     = "5.8.9"
}

variable "cilium_chart_version" {
  description = "Helm chart version for Cilium."
  type        = string
  default     = "1.16.19"
}
