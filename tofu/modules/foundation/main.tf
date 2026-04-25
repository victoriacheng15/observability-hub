terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 3.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = ">= 3.1"
    }
  }
}

# --- Shared Standards ---

locals {
  standards = yamldecode(file("${path.module}/../../../k3s/_standards.yaml")).homelab
}

# --- Namespaces ---

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

resource "kubernetes_namespace_v1" "argocd" {
  metadata {
    name = var.argocd_namespace
  }
}

resource "kubernetes_namespace_v1" "hardware_sim" {
  metadata {
    name = var.hardware_sim_namespace
  }
}

# --- CNI (Cilium) ---

resource "helm_release" "cilium" {
  name       = "cilium"
  repository = "https://helm.cilium.io/"
  chart      = "cilium"
  version    = var.cilium_chart_version
  namespace  = "kube-system"

  values = [
    yamlencode({
      # --- eBPF Datapath & Routing ---
      routingMode          = "native"
      kubeProxyReplacement = true
      bpf = {
        masquerade = true
      }
      ipv4NativeRoutingCIDR = "10.42.0.0/16"

      devices = ["eno1", "tailscale0", "lo"]

      k8sServiceHost = "10.0.0.245"
      k8sServicePort = "6443"

      hostFirewall = {
        enabled   = true
        auditMode = true
      }

      ipam = {
        mode = "cluster-pool"
        operator = {
          clusterPoolIPv4PodCIDRList = ["10.42.0.0/16"]
          clusterPoolIPv4MaskSize    = 24
        }
      }

      l7Proxy        = true
      enableL7Config = true
      mqtt = {
        enabled = true
      }

      hubble = {
        enabled             = true
        eventBufferCapacity = 32767
        metrics = {
          enabled = [
            "dns",
            "drop",
            "tcp",
            "flow",
            "port-distribution",
            "icmp",
            "httpV2:exemplars=true"
          ]
          serviceMonitor = {
            enabled = false
          }
        }
        relay = {
          enabled   = true
          resources = local.standards.resources.small
        }
        ui = {
          enabled   = true
          resources = local.standards.resources.small
          service = {
            type     = "NodePort"
            nodePort = 30067
          }
        }
      }

      prometheus = {
        enabled = true
        serviceMonitor = {
          enabled = false
        }
      }

      operator = {
        replicas = 1
        prometheus = {
          enabled = true
          serviceMonitor = {
            enabled = false
          }
        }
        resources = local.standards.resources.small
      }

      resources            = local.standards.resources.large
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
    })
  ]
}
