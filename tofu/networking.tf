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
      tunnel               = "disabled"
      kubeProxyReplacement = true
      bpf = {
        masquerade = true
      }
      ipv4NativeRoutingCIDR = "10.42.0.0/16" # K3s Default Pod CIDR

      # Explicit interface targeting to prevent management of wrong devices
      devices = ["eno1"]

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

      # --- L7 Visibility (MQTT) ---
      l7Proxy        = true
      enableL7Config = true
      mqtt = {
        enabled = true
      }

      # --- Hubble (Observability) ---
      hubble = {
        enabled = true
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
      resources            = local.standards.resources.standard
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
    })
  ]
}
# --- MQTT Broker (EMQX) ---

resource "helm_release" "emqx" {
  name       = "emqx"
  repository = "https://repos.emqx.io/charts"
  chart      = "emqx"
  version    = var.emqx_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    yamlencode({
      # Single-node optimizations
      replicaCount = 1

      emqxConfig = {
        "listeners.tcp.default.bind" = "1883"
      }

      # Standard Resource Limits & Standards
      resources            = local.standards.resources.medium
      revisionHistoryLimit = local.standards.deployment.revision_history_limit

      service = {
        type = "ClusterIP"
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- L7 MQTT Visibility Policy ---

resource "null_resource" "mqtt_visibility" {
  triggers = {
    manifest_sha1 = sha1(file("${path.module}/../k3s/opentelemetry/mqtt-visibility.yaml"))
  }

  provisioner "local-exec" {
    command = "kubectl apply -f ${path.module}/../k3s/opentelemetry/mqtt-visibility.yaml"
  }

  depends_on = [helm_release.emqx]
}
