output "otel_collector_endpoint" {
  value       = "opentelemetry-collector.${var.observability_namespace}.svc.cluster.local:4317"
  description = "The OTLP ingestion endpoint for the collector."
}
