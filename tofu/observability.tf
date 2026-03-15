# --- Shared Standards ---

locals {
  standards = yamldecode(file("${path.module}/../k3s/_standards.yaml")).homelab
}

# --- Metrics (Prometheus) ---

resource "helm_release" "prometheus" {
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "prometheus"
  version    = var.prometheus_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    file("${path.module}/../k3s/prometheus/values.yaml"),
    yamlencode({
      server = {
        revisionHistoryLimit = local.standards.deployment.revision_history_limit
        persistentVolume = {
          storageClass = local.standards.persistence.storage_class
          size         = local.standards.persistence.size
        }
        resources = local.standards.resources.large
        containerSecurityContext = {
          readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
          allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
          runAsNonRoot             = local.standards.security.container.run_as_non_root
          runAsUser                = 65534
          runAsGroup               = 65534
          capabilities = {
            drop = local.standards.security.container.capabilities_drop
          }
        }
        sidecarConfigReloader = {
          containerSecurityContext = {
            readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
            allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
            runAsNonRoot             = local.standards.security.container.run_as_non_root
            runAsUser                = 65534
            capabilities = {
              drop = local.standards.security.container.capabilities_drop
            }
          }
        }
        configmapReload = {
          prometheus = {
            containerSecurityContext = {
              readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
              allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
              runAsNonRoot             = local.standards.security.container.run_as_non_root
              runAsUser                = 65534
              capabilities = {
                drop = local.standards.security.container.capabilities_drop
              }
            }
          }
        }
      }
      kube-state-metrics = {
        revisionHistoryLimit = local.standards.deployment.revision_history_limit
        resources            = local.standards.resources.small
      }
      prometheus-node-exporter = {
        resources = local.standards.resources.small
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Energy Auditing (Kepler Native) ---

resource "kubernetes_service_account_v1" "kepler" {
  metadata {
    name      = "kepler"
    namespace = kubernetes_namespace_v1.observability.metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "kepler"
    }
  }
}

resource "kubernetes_cluster_role_v1" "kepler" {
  metadata {
    name = "kepler"
    labels = {
      "app.kubernetes.io/name" = "kepler"
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "nodes", "nodes/proxy", "nodes/metrics"]
    verbs      = ["get", "list", "watch"]
  }
}

resource "kubernetes_cluster_role_binding_v1" "kepler" {
  metadata {
    name = "kepler"
    labels = {
      "app.kubernetes.io/name" = "kepler"
    }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = kubernetes_cluster_role_v1.kepler.metadata[0].name
  }

  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account_v1.kepler.metadata[0].name
    namespace = kubernetes_namespace_v1.observability.metadata[0].name
  }
}

resource "kubernetes_config_map_v1" "kepler" {
  metadata {
    name      = "kepler"
    namespace = kubernetes_namespace_v1.observability.metadata[0].name
  }

  data = {
    "config.yaml" = yamlencode({
      exporter = {
        prometheus = {
          enabled = true
        }
      }
      host = {
        procfs = "/host/proc"
        sysfs  = "/host/sys"
      }
      log = {
        level = "debug"
      }
      monitor = {
        interval = "5s"
      }
      web = {
        listenAddresses = [":28282"]
      }
    })
  }
}

resource "kubernetes_service_v1" "kepler" {
  metadata {
    name      = "kepler"
    namespace = kubernetes_namespace_v1.observability.metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "kepler"
    }
  }

  spec {
    selector = {
      "app.kubernetes.io/name" = "kepler"
    }

    port {
      name        = "http"
      port        = 28282
      target_port = 28282
    }

    type = "ClusterIP"
  }
}

resource "kubernetes_daemon_set_v1" "kepler" {
  metadata {
    name      = "kepler"
    namespace = kubernetes_namespace_v1.observability.metadata[0].name
    labels = {
      "app.kubernetes.io/name" = "kepler"
    }
  }

  spec {
    selector {
      match_labels = {
        "app.kubernetes.io/name" = "kepler"
      }
    }

    template {
      metadata {
        labels = {
          "app.kubernetes.io/name" = "kepler"
        }
      }

      spec {
        service_account_name = kubernetes_service_account_v1.kepler.metadata[0].name
        host_pid             = true

        node_selector = {
          "kubernetes.io/os" = "linux"
        }

        toleration {
          operator = "Exists"
        }

        container {
          name              = "kepler"
          image             = "quay.io/sustainable_computing_io/kepler:latest"
          image_pull_policy = "IfNotPresent"

          security_context {
            privileged = true
          }

          command = ["/usr/bin/kepler"]
          args    = ["--config.file=/etc/kepler/config.yaml", "--kube.enable", "--kube.node-name=$(NODE_NAME)"]

          port {
            name           = "http"
            container_port = 28282
            protocol       = "TCP"
          }

          env {
            name = "NODE_NAME"
            value_from {
              field_ref {
                field_path = "spec.nodeName"
              }
            }
          }

          volume_mount {
            name       = "sysfs"
            mount_path = "/host/sys"
            read_only  = true
          }
          volume_mount {
            name       = "procfs"
            mount_path = "/host/proc"
            read_only  = true
          }
          volume_mount {
            name       = "cfm"
            mount_path = "/etc/kepler"
          }

          liveness_probe {
            http_get {
              path = "/metrics"
              port = "http"
            }
            initial_delay_seconds = 10
            period_seconds        = 60
          }
        }

        volume {
          name = "sysfs"
          host_path {
            path = "/sys"
          }
        }
        volume {
          name = "procfs"
          host_path {
            path = "/proc"
          }
        }
        volume {
          name = "cfm"
          config_map {
            name = kubernetes_config_map_v1.kepler.metadata[0].name
          }
        }
      }
    }
  }

  depends_on = [kubernetes_namespace_v1.observability]
}

resource "kubernetes_service_v1" "prometheus_thanos_grpc" {
  metadata {
    name      = "prometheus-thanos-grpc"
    namespace = kubernetes_namespace_v1.observability.metadata[0].name
    labels = {
      "app.kubernetes.io/name"      = "prometheus"
      "app.kubernetes.io/component" = "thanos-sidecar"
    }
  }

  spec {
    selector = {
      "app.kubernetes.io/name"      = "prometheus"
      "app.kubernetes.io/component" = "server"
      "app.kubernetes.io/instance"  = "prometheus"
    }

    port {
      name        = "grpc"
      port        = 10901
      target_port = 10901
    }

    type       = "ClusterIP"
    cluster_ip = "None" # Headless service for SRV discovery
  }
}

# --- Long-term Metrics (Thanos) ---

resource "helm_release" "thanos" {
  name       = "thanos"
  repository = "oci://registry-1.docker.io/bitnamicharts"
  chart      = "thanos"
  version    = var.thanos_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    file("${path.module}/../k3s/thanos/values.yaml"),
    yamlencode({
      query = {
        extraFlags = ["--endpoint=prometheus-thanos-grpc.observability.svc.cluster.local:10901"]
      }
      storegateway = {
        persistence = {
          storageClass = local.standards.persistence.storage_class
          size         = "2Gi" # Preserve existing override
        }
        resources = local.standards.resources.medium
        podSecurityContext = {
          enabled      = true
          runAsNonRoot = local.standards.security.pod.run_as_non_root
          fsGroup      = local.standards.security.pod.fs_group
          runAsUser    = local.standards.security.pod.run_as_user
          runAsGroup   = local.standards.security.pod.run_as_group
        }
        containerSecurityContext = {
          enabled                  = true
          readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
          allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
          capabilities = {
            drop = local.standards.security.container.capabilities_drop
          }
        }
      }
      compactor = {
        persistence = {
          storageClass = local.standards.persistence.storage_class
          size         = "2Gi" # Preserve existing override
        }
        resources = local.standards.resources.medium
        podSecurityContext = {
          enabled      = true
          runAsNonRoot = local.standards.security.pod.run_as_non_root
          fsGroup      = local.standards.security.pod.fs_group
          runAsUser    = local.standards.security.pod.run_as_user
          runAsGroup   = local.standards.security.pod.run_as_group
        }
        containerSecurityContext = {
          enabled                  = true
          readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
          allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
          capabilities = {
            drop = local.standards.security.container.capabilities_drop
          }
        }
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Logs (Loki) ---

resource "helm_release" "loki" {
  name       = "loki"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki"
  version    = var.loki_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    file("${path.module}/../k3s/loki/values.yaml"),
    yamlencode({
      loki = {
        persistence = {
          storageClass = local.standards.persistence.storage_class
          size         = local.standards.persistence.size
        }
      }
      singleBinary = {
        persistence = {
          storageClass = local.standards.persistence.storage_class
          size         = local.standards.persistence.size
        }
        resources = local.standards.resources.large
        podSecurityContext = {
          runAsNonRoot = local.standards.security.pod.run_as_non_root
          fsGroup      = local.standards.security.pod.fs_group
          runAsUser    = local.standards.security.pod.run_as_user
          runAsGroup   = local.standards.security.pod.run_as_group
        }
        containerSecurityContext = {
          readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
          allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
          capabilities = {
            drop = local.standards.security.container.capabilities_drop
          }
        }
      }
      gateway = {
        deploymentStrategy = {
          type = "Recreate"
        }
        affinity  = null
        resources = local.standards.resources.small
        containerSecurityContext = {
          readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
          allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
          capabilities = {
            drop = local.standards.security.container.capabilities_drop
          }
        }
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Traces (Tempo) ---

resource "helm_release" "tempo" {
  name       = "tempo"
  repository = "https://grafana-community.github.io/helm-charts"
  chart      = "tempo"
  version    = var.tempo_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    file("${path.module}/../k3s/tempo/values.yaml"),
    yamlencode({
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
      persistence = {
        storageClassName = local.standards.persistence.storage_class
        size             = local.standards.persistence.size
      }
      tempo = {
        resources = local.standards.resources.large
        securityContext = {
          readOnlyRootFilesystem = local.standards.security.container.read_only_root_fs
        }
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Signal Processing (OpenTelemetry) ---

resource "helm_release" "opentelemetry_collector" {
  name       = "opentelemetry-collector"
  repository = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart      = "opentelemetry-collector"
  version    = var.otel_collector_chart_version
  namespace  = kubernetes_namespace_v1.observability.metadata[0].name

  values = [
    file("${path.module}/../k3s/opentelemetry/values.yaml"),
    yamlencode({
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
      resources            = local.standards.resources.medium
      podSecurityContext = {
        runAsNonRoot = local.standards.security.pod.run_as_non_root
        fsGroup      = local.standards.security.pod.fs_group
        runAsUser    = local.standards.security.pod.run_as_user
        runAsGroup   = local.standards.security.pod.run_as_group
      }
      securityContext = {
        readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
        allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
        capabilities = {
          drop = local.standards.security.container.capabilities_drop
        }
      }
    })
  ]

  depends_on = [kubernetes_namespace_v1.observability]
}

# --- Visualization (Grafana) ---

resource "helm_release" "grafana" {
  name       = "grafana"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "grafana"
  version    = var.grafana_chart_version
  namespace  = kubernetes_namespace_v1.hub.metadata[0].name

  values = [
    file("${path.module}/../k3s/grafana/values.yaml"),
    yamlencode({
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
      persistence = {
        storageClass = local.standards.persistence.storage_class
        size         = local.standards.persistence.size
      }
      podSecurityContext = {
        runAsNonRoot = false
        runAsUser    = 0
        runAsGroup   = 0
        fsGroup      = 472
      }
      containerSecurityContext = {
        readOnlyRootFilesystem   = local.standards.security.container.read_only_root_fs
        allowPrivilegeEscalation = local.standards.security.container.allow_privilege_escalation
        runAsNonRoot             = true
        runAsUser                = 472
        runAsGroup               = 472
        capabilities = {
          drop = local.standards.security.container.capabilities_drop
        }
      }
      initChownData = {
        securityContext = {
          readOnlyRootFilesystem   = local.standards.exceptions.grafana.init_chown.container_read_only_root_fs
          allowPrivilegeEscalation = local.standards.exceptions.grafana.init_chown.allow_privilege_escalation
          runAsNonRoot             = false
          runAsUser                = 0
          capabilities = {
            add  = local.standards.exceptions.grafana.init_chown.add_capabilities
            drop = local.standards.security.container.capabilities_drop
          }
        }
      }
      resources = local.standards.resources.medium
    })
  ]

  depends_on = [kubernetes_namespace_v1.hub]
}

resource "grafana_folder" "observability" {
  title = "Observability"
}

resource "grafana_dashboard" "dashboards" {
  for_each = fileset("${path.module}/../k3s/grafana/dashboards", "*.json")

  folder      = grafana_folder.observability.id
  config_json = file("${path.module}/../k3s/grafana/dashboards/${each.value}")
  overwrite   = true
}

# --- Analytics (Resource-to-Value Engine) ---

resource "kubernetes_daemon_set_v1" "analytics" {
  metadata {
    name      = "analytics"
    namespace = kubernetes_namespace_v1.hub.metadata[0].name
    labels = {
      "app.kubernetes.io/name"    = "analytics"
      "app.kubernetes.io/feature" = "analytics-engine"
    }
  }

  spec {
    selector {
      match_labels = {
        "app.kubernetes.io/name" = "analytics"
      }
    }

    template {
      metadata {
        labels = {
          "app.kubernetes.io/name" = "analytics"
        }
      }

      spec {
        host_network = true
        dns_policy   = "ClusterFirstWithHostNet"

        container {
          name              = "analytics"
          image             = "analytics:v0.1.0"
          image_pull_policy = "IfNotPresent"

          # Observability Endpoints (FQDN)
          env {
            name  = "THANOS_URL"
            value = "http://thanos-query.observability.svc.cluster.local:9090"
          }
          env {
            name  = "OTEL_EXPORTER_OTLP_ENDPOINT"
            value = "opentelemetry.observability.svc.cluster.local:4317"
          }

          # Database Credentials
          env {
            name  = "DB_HOST"
            value = "postgres-hub-rw.databases.svc.cluster.local"
          }
          env {
            name  = "DB_PORT"
            value = "5432"
          }
          env {
            name  = "DB_USER"
            value = "server"
          }
          env {
            name  = "DB_NAME"
            value = "homelab"
          }
          env {
            name = "SERVER_DB_PASSWORD"
            value_from {
              secret_key_ref {
                name = "postgres-secret"
                key  = "server-db-password"
              }
            }
          }

          resources {
            requests = {
              cpu    = "100m"
              memory = "200Mi"
            }
            limits = {
              cpu    = "100m"
              memory = "400Mi"
            }
          }

          volume_mount {
            name       = "tailscale-sock"
            mount_path = "/var/run/tailscale/tailscaled.sock"
            read_only  = true
          }
          volume_mount {
            name       = "host-hostname"
            mount_path = "/etc/host_hostname"
            read_only  = true
          }
          volume_mount {
            name       = "host-os-release"
            mount_path = "/etc/host_os-release"
            read_only  = true
          }
        }

        volume {
          name = "tailscale-sock"
          host_path {
            path = "/var/run/tailscale/tailscaled.sock"
            type = "Socket"
          }
        }
        volume {
          name = "host-hostname"
          host_path {
            path = "/etc/hostname"
            type = "File"
          }
        }
        volume {
          name = "host-os-release"
          host_path {
            path = "/etc/os-release"
            type = "File"
          }
        }
      }
    }
  }

  depends_on = [kubernetes_namespace_v1.hub]
}
