resource "kubernetes_storage_class_v1" "local_path_retain" {
  metadata {
    name = "local-path-retain"
  }

  storage_provisioner = "rancher.io/local-path"
  reclaim_policy      = "Retain"
  volume_binding_mode = "WaitForFirstConsumer"
}
