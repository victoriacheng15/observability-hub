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

# --- Metrics (Prometheus) ---

data "kubernetes_service_v1" "kube_dns" {
  metadata {
    name      = "kube-dns"
    namespace = "kube-system"
  }
}

resource "helm_release" "prometheus" {
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "prometheus"
  version    = var.prometheus_chart_version
  namespace  = var.observability_namespace

  values = [
    file("${path.module}/../../../k3s/base/infra/prometheus/values.yaml"),
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
}

# --- Energy Auditing (Kepler Native) ---

resource "kubernetes_service_account_v1" "kepler" {
  metadata {
    name      = "kepler"
    namespace = var.observability_namespace
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
    namespace = var.observability_namespace
  }
}

resource "kubernetes_config_map_v1" "kepler" {
  metadata {
    name      = "kepler"
    namespace = var.observability_namespace
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
    namespace = var.observability_namespace
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
    namespace = var.observability_namespace
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
}

# --- Logs (Loki) ---

resource "helm_release" "loki" {
  name       = "loki"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki"
  version    = var.loki_chart_version
  namespace  = var.observability_namespace

  values = [
    file("${path.module}/../../../k3s/base/infra/loki/values.yaml"),
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
        resources = local.standards.resources.standard
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
        resources = local.standards.resources.medium
        nginxConfig = {
          resolver = data.kubernetes_service_v1.kube_dns.spec[0].cluster_ip
        }
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
}

# --- Traces (Tempo) ---

resource "helm_release" "tempo" {
  name       = "tempo"
  repository = "https://grafana-community.github.io/helm-charts"
  chart      = "tempo"
  version    = var.tempo_chart_version
  namespace  = var.observability_namespace

  values = [
    file("${path.module}/../../../k3s/base/infra/tempo/values.yaml"),
    yamlencode({
      revisionHistoryLimit = local.standards.deployment.revision_history_limit
      serviceAccount = {
        create = false
        name   = "tempo"
      }
      automountServiceAccountToken = false
      persistence = {
        storageClassName = local.standards.persistence.storage_class
        size             = local.standards.persistence.size
      }
      tempo = {
        resources = local.standards.resources.standard
        securityContext = {
          readOnlyRootFilesystem = local.standards.security.container.read_only_root_fs
        }
      }
    })
  ]
}

# --- Signal Processing (OpenTelemetry) ---

resource "helm_release" "opentelemetry_collector" {
  name       = "opentelemetry-collector"
  repository = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart      = "opentelemetry-collector"
  version    = var.otel_collector_chart_version
  namespace  = var.observability_namespace

  values = [
    file("${path.module}/../../../k3s/base/infra/opentelemetry/values.yaml"),
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
}
