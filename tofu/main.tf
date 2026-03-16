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
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
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

# --- Shared Standards ---

locals {
  standards = yamldecode(file("${path.module}/../k3s/_standards.yaml")).homelab
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

resource "kubernetes_namespace_v1" "hub" {
  metadata {
    name = var.hub_namespace
  }
}

moved {
  from = kubernetes_namespace.observability
  to   = kubernetes_namespace_v1.observability
}
