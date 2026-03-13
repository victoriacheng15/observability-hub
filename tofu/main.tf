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
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }

  backend "azurerm" {}
}

# --- Providers ---

provider "azurerm" {
  features {}
}

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

variable "databases_namespace" {
  description = "Namespace for all database and persistence services."
  type        = string
  default     = "databases"
}

# --- Namespace ---

resource "kubernetes_namespace_v1" "observability" {
  metadata {
    name = var.observability_namespace
  }
}

resource "kubernetes_namespace_v1" "databases" {
  metadata {
    name = var.databases_namespace
  }
}

moved {
  from = kubernetes_namespace.observability
  to   = kubernetes_namespace_v1.observability
}
