variable "observability_namespace" {
  description = "Namespace for all observability services."
  type        = string
}

variable "databases_namespace" {
  description = "Namespace for all database and persistence services."
  type        = string
}

variable "hub_namespace" {
  description = "Namespace for analytical and hub-facing services."
  type        = string
}

variable "argocd_namespace" {
  description = "Namespace for ArgoCD GitOps controller."
  type        = string
}

variable "hardware_sim_namespace" {
  description = "Namespace for hardware simulation and chaos experiments."
  type        = string
}

variable "cilium_chart_version" {
  description = "Helm chart version for Cilium."
  type        = string
}
