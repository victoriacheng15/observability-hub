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
      }
      # Disable unused components for a leaner footprint
      notifications = { enabled = false }
      dex           = { enabled = false }
    })
  ]

  depends_on = [kubernetes_namespace_v1.argocd]
}

# --- Root Application (App-of-Apps) Bootstrap ---
# We use a null_resource to apply the manifest because the ArgoCD Application CRD
# is only available AFTER the Helm chart is installed. Using a local variable
# avoids external file dependencies.

resource "null_resource" "argocd_root_app" {
  triggers = {
    # Re-apply if the manifest structure changes
    manifest_sha1 = sha1(local.root_app_manifest)
  }

  provisioner "local-exec" {
    command = "echo '${local.root_app_manifest}' | kubectl apply -f -"
  }

  depends_on = [helm_release.argocd]
}

locals {
  root_app_manifest = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "root-app"
      namespace = var.argocd_namespace
      finalizers = [
        "resources-finalizer.argocd.argoproj.io"
      ]
    }
    spec = {
      project = "default"
      source = {
        repoURL        = "https://github.com/victoriacheng15/observability-hub.git"
        targetRevision = "HEAD"
        path           = "k3s"
        directory = {
          recurse = true
          # Exclude non-manifest files to prevent sync errors
          exclude = "{README.md,*.json,*.tar.gz,*.sh}"
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = var.argocd_namespace
      }
      syncPolicy = {
        automated = {
          prune    = true
          selfHeal = true
        }
        retry = {
          limit = 5
          backoff = {
            duration    = "5s"
            factor      = 2
            maxDuration = "3m"
          }
        }
        syncOptions = [
          "CreateNamespace=true",
          "PruneLast=true",
          "ApplyOutOfSyncOnly=true"
        ]
      }
    }
  })
}
