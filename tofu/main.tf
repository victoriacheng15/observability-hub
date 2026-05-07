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
}

# --- Persistence ---

module "persistence" {
  source = "./modules/persistence"

  emqx_chart_version         = var.emqx_chart_version
  observability_namespace    = module.foundation.observability_namespace
  databases_namespace        = module.foundation.databases_namespace
  azure_storage_account_name = module.storage.azure_storage_account_name
}

# --- Observability ---

module "observability" {
  source = "./modules/observability"

  prometheus_chart_version     = var.prometheus_chart_version
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
