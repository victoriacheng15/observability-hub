terraform {
  required_version = ">= 1.6.0"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 3.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 3.1"
    }
    grafana = {
      source  = "grafana/grafana"
      version = "~> 4.27"
    }
  }

  # backend "s3" {
  #   bucket                      = "tofu-state"
  #   key                         = "observability-hub/terraform.tfstate"
  #   region                      = "minio"
  #   endpoint                    = "http://<minio-node-ip>:9000"
  #   skip_credentials_validation = true
  #   skip_metadata_api_check     = true
  #   skip_region_validation      = true
  #   force_path_style            = true
  # }
}

# --- Providers ---

provider "kubernetes" {
  config_path    = var.kubeconfig_path
  config_context = "default"
}

provider "helm" {
  kubernetes = {
    config_path    = var.kubeconfig_path
    config_context = "default"
  }
}

data "kubernetes_secret_v1" "grafana_admin" {
  metadata {
    name      = "grafana-admin-secret"
    namespace = var.observability_namespace
  }
}

provider "grafana" {
  url  = "http://localhost:30000"
  auth = try("${data.kubernetes_secret_v1.grafana_admin.data["admin-user"]}:${data.kubernetes_secret_v1.grafana_admin.data["admin-password"]}", "admin:admin")
}

# --- Variables ---

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

# --- Namespace ---

resource "kubernetes_namespace_v1" "observability" {
  metadata {
    name = var.observability_namespace
  }
}

moved {
  from = kubernetes_namespace.observability
  to   = kubernetes_namespace_v1.observability
}
