variable "kubeconfig_path" {
  description = "Path to the kubeconfig file."
  type        = string
  default     = "~/.kube/config"
}

variable "observability_namespace" {
  description = "Namespace for all observability services."
  type        = string
  default     = "observability"
}
