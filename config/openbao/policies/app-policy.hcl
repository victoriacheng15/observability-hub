# Grant read access to all observability-hub secrets
path "secret/data/observability-hub/*" {
  capabilities = ["read"]
}
