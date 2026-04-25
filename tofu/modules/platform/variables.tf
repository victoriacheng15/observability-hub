variable "grafana_chart_version" {
  description = "Helm chart version for Grafana."
  type        = string
}

variable "argocd_chart_version" {
  description = "Helm chart version for ArgoCD."
  type        = string
}

variable "grafana_discord_webhook_url" {
  description = "Discord webhook URL for Grafana alerts."
  type        = string
  sensitive   = true
  default     = null
}

variable "hub_namespace" {
  description = "Namespace for analytical and hub-facing services."
  type        = string
}

variable "argocd_namespace" {
  description = "Namespace for ArgoCD GitOps controller."
  type        = string
}
