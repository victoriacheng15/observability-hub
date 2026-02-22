# Collectors Service Architecture

The Collectors Service (`k3s/collectors/`) is a **Host Telemetry Collector** deployed as a Kubernetes DaemonSet. Its primary responsibility is to gather host-level telemetry, including retrieving host-level metrics (CPU, RAM, disk, network) from Prometheus and collecting Tailscale status and data. This collected data is then forwarded to the central **OpenTelemetry Collector** for further processing and ingestion into the observability backend.

## 🎯 Objective

To provide a unified mechanism for host-level telemetry collection, ensuring critical operational data from the underlying infrastructure and network services (like Tailscale) is captured and integrated into the observability platform.

## 🧩 Component Details

- **Type**: Kubernetes DaemonSet.
- **Source**: Prometheus (for host metrics), Tailscale API (for status/data).
- **Destination**: OpenTelemetry Collector (via OTLP).
- **Deployment**: Managed as a Helm chart within `k3s/collectors/`.

## ⚙️ Logic Flow

1. **Deployment**: The service runs as a DaemonSet, ensuring an instance is active on each relevant host in the Kubernetes cluster.
2. **Host Metrics Collection**: Retrieves host-level metrics (CPU, RAM, disk, network) from Prometheus.
3. **Tailscale Status/Data**: Gathers operational status and data from Tailscale.
4. **Forwarding**: Formats the collected metrics and traces (where applicable) into OTLP and forwards them to the **OpenTelemetry Collector**.

## 🔭 Observability Implementation

The Collectors are an integral part of the host-level observability strategy.

- **Output**: Emits collected metrics and traces in OpenTelemetry Protocol (OTLP) format.
- **Integration**: Seamlessly integrates with the OpenTelemetry Collector as an upstream source of host telemetry.
- **Data Types**: Primarily deals with metrics and traces related to host performance and network status.
