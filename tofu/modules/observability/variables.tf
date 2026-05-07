variable "prometheus_chart_version" {
  description = "Helm chart version for Prometheus."
  type        = string
}

variable "loki_chart_version" {
  description = "Helm chart version for Loki."
  type        = string
}

variable "tempo_chart_version" {
  description = "Helm chart version for Tempo."
  type        = string
}

variable "otel_collector_chart_version" {
  description = "Helm chart version for OpenTelemetry Collector."
  type        = string
}

variable "observability_namespace" {
  description = "Namespace for observability services."
  type        = string
}
