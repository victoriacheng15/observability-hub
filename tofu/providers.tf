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

  # Step 9: MinIO state backend - uncomment AFTER MinIO pod is confirmed healthy (Step 4)
  # This stores OpenTofu's own .tfstate file inside MinIO, separate from the MinIO pod itself.
  # backend "s3" {
  #   bucket                      = "tofu-state"
  #   key                         = "observability-hub/terraform.tfstate"
  #   region                      = "minio"          # dummy value - required by S3 SDK but ignored
  #   endpoint                    = "http://<minio-node-ip>:9000"
  #   skip_credentials_validation = true
  #   skip_metadata_api_check     = true
  #   skip_region_validation      = true
  #   force_path_style            = true
  # }
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
