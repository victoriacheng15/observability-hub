# --- ArgoCD GitOps Controller ---

resource "helm_release" "argocd" {
  name       = "argocd"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = var.argocd_chart_version
  namespace  = kubernetes_namespace_v1.argocd.metadata[0].name

  values = [
    yamlencode({
      global = {
        domain               = "argocd.observability-hub.home"
        revisionHistoryLimit = local.standards.deployment.revision_history_limit
        image = {
          imagePullPolicy = local.standards.deployment.image_pull_policy
        }
      }
      configs = {
        cm = {
          "application.instanceLabelKey"       = "argocd.argoproj.io/instance"
          "application.resourceTrackingMethod" = "annotation+label"
        }
      }
      server = {
        extraArgs = ["--insecure"]
        service = {
          type          = "NodePort"
          nodePortHttp  = 30088
          nodePortHttps = 30443
        }
        resources = local.standards.resources.standard
      }
      controller = {
        resources = local.standards.resources.large
      }
      repoServer = {
        resources = local.standards.resources.medium
      }
      applicationSet = {
        resources = local.standards.resources.small
      }
      redis = {
        resources = local.standards.resources.small
        persistence = {
          enabled      = true
          storageClass = local.standards.persistence.storage_class
          size         = "2Gi" # Specific cache override
        }
        # Identity Handshake: Use the ServiceAccount managed in k3s/base/rbac
        serviceAccount = {
          create = false
          name   = "argocd-redis"
        }
        automountServiceAccountToken = false
      }
      # Disable unused components for a leaner footprint
      notifications = { enabled = false }
      dex           = { enabled = false }
    })
  ]

  depends_on = [kubernetes_namespace_v1.argocd]
}
