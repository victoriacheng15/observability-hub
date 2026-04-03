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
      ipv4NativeRoutingCIDR = "10.42.0.0/16" # K3s Default Pod CIDR

      # Explicit interface targeting to prevent management of wrong devices
      devices = ["eno1", "tailscale0", "lo"]

      # --- Bootstrap Stability ---
      # Required when kube-proxy is replaced to ensure agent connectivity
      k8sServiceHost = "10.0.0.245"
      k8sServicePort = "6443"

      # --- SSH Lockout Prevention (Host Firewall) ---
      # auditMode: true = Log drops instead of blocking (Fail-Open Safety)
      hostFirewall = {
        enabled   = true
        auditMode = true
      }

      # --- IPAM (Address Management) ---
      ipam = {
        mode = "cluster-pool"
        operator = {
          clusterPoolIPv4PodCIDRList = ["10.42.0.0/16"]
          clusterPoolIPv4MaskSize    = 24
        }
      }

      # --- L7 Visibility (MQTT/HTTP) ---
      l7Proxy        = true
      enableL7Config = true
      mqtt = {
        enabled = true
      }

      # --- Hubble (Observability) ---
      hubble = {
        enabled = true
        # Ring buffer capacity (default: 4095)
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
          # Disable ServiceMonitor as Prometheus Operator is not present
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

      # --- Integration ---
      prometheus = {
        enabled = true
        serviceMonitor = {
          enabled = false
        }
      }

      operator = {
        replicas = 1 # Single-node optimization
        prometheus = {
          enabled = true
          serviceMonitor = {
            enabled = false
          }
        }
        resources = local.standards.resources.small
      }

      # Standard Resource Limits & Standards
      resources            = local.standards.resources.large
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
    })
  ]
}
