# --- Foundation ---

module "foundation" {
  source = "./modules/foundation"

  observability_namespace = var.observability_namespace
  databases_namespace     = var.databases_namespace
  hub_namespace           = var.hub_namespace
  argocd_namespace        = var.argocd_namespace
  hardware_sim_namespace  = var.hardware_sim_namespace
  cilium_chart_version    = var.cilium_chart_version
}

# --- Storage ---

module "storage" {
  source = "./modules/storage"

  azurerm_storage_account_name = var.azurerm_storage_account_name
  azurerm_resource_group_name  = var.azurerm_resource_group_name
  minio_chart_version          = var.minio_chart_version
  databases_namespace          = module.foundation.databases_namespace
}

# --- Persistence ---

module "persistence" {
  source = "./modules/persistence"

  emqx_chart_version            = var.emqx_chart_version
  cnpg_operator_chart_version   = var.cnpg_operator_chart_version
  postgres_config               = var.postgres_config
  postgres_node_port            = var.postgres_node_port
  databases_namespace           = module.foundation.databases_namespace
  observability_namespace       = module.foundation.observability_namespace
  azure_storage_account_name    = module.storage.azure_storage_account_name
  local_path_storage_class_name = module.storage.local_path_storage_class_name
}

# --- Observability ---

module "observability" {
  source = "./modules/observability"

  prometheus_chart_version     = var.prometheus_chart_version
  thanos_chart_version         = var.thanos_chart_version
  loki_chart_version           = var.loki_chart_version
  tempo_chart_version          = var.tempo_chart_version
  otel_collector_chart_version = var.otel_collector_chart_version
  observability_namespace      = module.foundation.observability_namespace
}

# --- Platform ---

module "platform" {
  source = "./modules/platform"

  grafana_chart_version       = var.grafana_chart_version
  argocd_chart_version        = var.argocd_chart_version
  grafana_discord_webhook_url = var.grafana_discord_webhook_url
  hub_namespace               = module.foundation.hub_namespace
  argocd_namespace            = module.foundation.argocd_namespace
}

# --- State Migration (Moved Blocks - Commented out after successful apply) ---

/*
moved {
  from = kubernetes_namespace_v1.observability
  to   = module.foundation.kubernetes_namespace_v1.observability
}

moved {
  from = kubernetes_namespace_v1.databases
  to   = module.foundation.kubernetes_namespace_v1.databases
}

moved {
  from = kubernetes_namespace_v1.hub
  to   = module.foundation.kubernetes_namespace_v1.hub
}

moved {
  from = kubernetes_namespace_v1.argocd
  to   = module.foundation.kubernetes_namespace_v1.argocd
}

moved {
  from = kubernetes_namespace_v1.hardware_sim
  to   = module.foundation.kubernetes_namespace_v1.hardware_sim
}

moved {
  from = helm_release.cilium
  to   = module.foundation.helm_release.cilium
}

moved {
  from = azurerm_storage_container.pg_backup
  to   = module.storage.azurerm_storage_container.pg_backup
}

moved {
  from = azurerm_storage_management_policy.pg_backup_lifecycle
  to   = module.storage.azurerm_storage_management_policy.pg_backup_lifecycle
}

moved {
  from = kubernetes_storage_class_v1.local_path_retain
  to   = module.storage.kubernetes_storage_class_v1.local_path_retain
}

moved {
  from = helm_release.minio
  to   = module.storage.helm_release.minio
}

moved {
  from = helm_release.emqx
  to   = module.persistence.helm_release.emqx
}

moved {
  from = helm_release.cnpg_operator
  to   = module.persistence.helm_release.cnpg_operator
}

moved {
  from = kubernetes_manifest.postgres_cluster
  to   = module.persistence.kubernetes_manifest.postgres_cluster
}

moved {
  from = kubernetes_manifest.postgres_backup_schedule
  to   = module.persistence.kubernetes_manifest.postgres_backup_schedule
}

moved {
  from = kubernetes_service_v1.postgres_nodeport
  to   = module.persistence.kubernetes_service_v1.postgres_nodeport
}

moved {
  from = helm_release.prometheus
  to   = module.observability.helm_release.prometheus
}

moved {
  from = kubernetes_service_account_v1.kepler
  to   = module.observability.kubernetes_service_account_v1.kepler
}

moved {
  from = kubernetes_cluster_role_v1.kepler
  to   = module.observability.kubernetes_cluster_role_v1.kepler
}

moved {
  from = kubernetes_cluster_role_binding_v1.kepler
  to   = module.observability.kubernetes_cluster_role_binding_v1.kepler
}

moved {
  from = kubernetes_config_map_v1.kepler
  to   = module.observability.kubernetes_config_map_v1.kepler
}

moved {
  from = kubernetes_service_v1.kepler
  to   = module.observability.kubernetes_service_v1.kepler
}

moved {
  from = kubernetes_daemon_set_v1.kepler
  to   = module.observability.kubernetes_daemon_set_v1.kepler
}

moved {
  from = kubernetes_service_v1.prometheus_thanos_grpc
  to   = module.observability.kubernetes_service_v1.prometheus_thanos_grpc
}

moved {
  from = helm_release.thanos
  to   = module.observability.helm_release.thanos
}

moved {
  from = helm_release.loki
  to   = module.observability.helm_release.loki
}

moved {
  from = helm_release.tempo
  to   = module.observability.helm_release.tempo
}

moved {
  from = helm_release.opentelemetry_collector
  to   = module.observability.helm_release.opentelemetry_collector
}

moved {
  from = helm_release.grafana
  to   = module.platform.helm_release.grafana
}

moved {
  from = grafana_folder.observability
  to   = module.platform.grafana_folder.observability
}

moved {
  from = grafana_dashboard.dashboards
  to   = module.platform.grafana_dashboard.dashboards
}

moved {
  from = grafana_contact_point.discord
  to   = module.platform.grafana_contact_point.discord
}

moved {
  from = grafana_notification_policy.default
  to   = module.platform.grafana_notification_policy.default
}

moved {
  from = grafana_rule_group.loki_log_errors
  to   = module.platform.grafana_rule_group.loki_log_errors
}

moved {
  from = helm_release.argocd
  to   = module.platform.helm_release.argocd
}
*/
