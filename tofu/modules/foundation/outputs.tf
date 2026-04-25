output "observability_namespace" {
  value       = kubernetes_namespace_v1.observability.metadata[0].name
  description = "The name of the observability namespace."
}

output "databases_namespace" {
  value       = kubernetes_namespace_v1.databases.metadata[0].name
  description = "The name of the databases namespace."
}

output "hub_namespace" {
  value       = kubernetes_namespace_v1.hub.metadata[0].name
  description = "The name of the hub namespace."
}

output "argocd_namespace" {
  value       = kubernetes_namespace_v1.argocd.metadata[0].name
  description = "The name of the argocd namespace."
}

output "hardware_sim_namespace" {
  value       = kubernetes_namespace_v1.hardware_sim.metadata[0].name
  description = "The name of the hardware simulation namespace."
}
