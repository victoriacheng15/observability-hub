# Hardware Simulation Validation Runbook

This runbook documents how to prove the synthetic hardware communication lab works end to end:

```text
sensor -> MQTT broker -> mqtt-ingestor -> logs/metrics -> Grafana
```

Use this guide when validating a new rollout, checking a regression, or demonstrating the behavior of the lab.

## Preconditions

- The `sensor-fleet`, `chaos-controller`, and `mqtt-ingestor` workloads are running in either `hardware-sim` or `dev-hardware-sim`.
- EMQX is reachable at `emqx.observability.svc.cluster.local:1883`.
- Grafana dashboards are provisioned from `k3s/base/infra/grafana/dashboards/edge-simulation.json`.
- Prometheus is scraping OTel-exported hardware metrics.
- Loki is receiving logs from `mqtt-ingestor`.

## Quick Runtime Checks

Check the workloads:

```bash
kubectl get deploy,statefulset,pods -n hardware-sim
kubectl get deploy,statefulset,pods -n dev-hardware-sim
```

Check the ingestor logs:

```bash
kubectl logs -n hardware-sim deploy/mqtt-ingestor --tail=50
kubectl logs -n dev-hardware-sim deploy/mqtt-ingestor --tail=50
```

Healthy steady-state logs should include:

- `mqtt_ingestor_started`
- `mqtt_subscription_established`
- `mqtt_message_received`

## Manual MQTT Publish Workflow

These examples use an ephemeral `mosquitto_pub` client pod. The examples target the broker service directly:

```text
emqx.observability.svc.cluster.local:1883
```

Use one namespace at a time for validation. The examples below use a manual device ID of `manual-lab-001`.
For every example except the explicit stale-message check, use a current UTC timestamp at publish time.

### Valid Telemetry

Publish a well-formed message:

```bash
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/manual-lab-001/telemetry \
  -m "{\"schema_version\":\"1\",\"sensor_id\":\"manual-lab-001\",\"device_id\":\"manual-lab-001\",\"firmware_version\":\"learning-lab-0.1.0\",\"device_state\":\"running\",\"sequence_number\":1,\"telemetry_topic\":\"devices/manual-lab-001/telemetry\",\"temperature\":41.2,\"voltage\":3.29,\"current\":0.11,\"power_usage\":0.36,\"rssi\":-58.0,\"snr\":9.1,\"packet_loss_percent\":0.2,\"free_heap\":182000,\"loop_time_ms\":6.4,\"uptime_seconds\":120,\"reboot_reason\":\"power_on\",\"timestamp\":\"$NOW\"}"
```

Expected outcome:

- Log: `mqtt_message_received`
- Metrics: `hardware_telemetry_messages_total` increments
- Dashboard:
  - `Active Devices (5m)` increases or keeps the device visible
  - `Lifecycle State Activity` shows `running`
  - radio and runtime panels include the sample in their rolling averages

### Malformed Payload

Publish invalid JSON:

```bash
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/manual-lab-001/telemetry \
  -m 'not-json'
```

Expected outcome:

- Log: `mqtt_payload_invalid`
- Metrics: `hardware_telemetry_invalid_messages_total` increments
- Dashboard:
  - `Invalid Payloads (1h)` increases
  - `Sequence And Payload Health` shows invalid-message activity

### Duplicate Sequence

Publish two identical sequence numbers:

```bash
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/manual-lab-001/telemetry \
  -m "{\"schema_version\":\"1\",\"sensor_id\":\"manual-lab-001\",\"device_id\":\"manual-lab-001\",\"firmware_version\":\"learning-lab-0.1.0\",\"device_state\":\"running\",\"sequence_number\":10,\"telemetry_topic\":\"devices/manual-lab-001/telemetry\",\"temperature\":41.0,\"voltage\":3.28,\"current\":0.11,\"power_usage\":0.36,\"rssi\":-58.0,\"snr\":9.0,\"packet_loss_percent\":0.1,\"free_heap\":181000,\"loop_time_ms\":6.3,\"uptime_seconds\":200,\"reboot_reason\":\"power_on\",\"timestamp\":\"$NOW\"}"
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/manual-lab-001/telemetry \
  -m "{\"schema_version\":\"1\",\"sensor_id\":\"manual-lab-001\",\"device_id\":\"manual-lab-001\",\"firmware_version\":\"learning-lab-0.1.0\",\"device_state\":\"running\",\"sequence_number\":10,\"telemetry_topic\":\"devices/manual-lab-001/telemetry\",\"temperature\":41.0,\"voltage\":3.28,\"current\":0.11,\"power_usage\":0.36,\"rssi\":-58.0,\"snr\":9.0,\"packet_loss_percent\":0.1,\"free_heap\":181000,\"loop_time_ms\":6.3,\"uptime_seconds\":201,\"reboot_reason\":\"power_on\",\"timestamp\":\"$NOW\"}"
```

Expected outcome:

- Log: `mqtt_duplicate_sequence_detected`
- Metrics: `hardware_telemetry_duplicates_total` increments
- Dashboard: `Sequence And Payload Health` shows duplicate activity

### Stale Message

Publish a valid message with an old timestamp:

```bash
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/manual-lab-001/telemetry \
  -m '{"schema_version":"1","sensor_id":"manual-lab-001","device_id":"manual-lab-001","firmware_version":"learning-lab-0.1.0","device_state":"running","sequence_number":11,"telemetry_topic":"devices/manual-lab-001/telemetry","temperature":40.8,"voltage":3.27,"current":0.10,"power_usage":0.34,"rssi":-60.0,"snr":8.8,"packet_loss_percent":0.3,"free_heap":180500,"loop_time_ms":6.1,"uptime_seconds":205,"reboot_reason":"power_on","timestamp":"2026-04-24T10:00:00Z"}'
```

Expected outcome:

- Log: `device_message_stale`
- Metrics: `hardware_telemetry_message_age_milliseconds_*` reflects the older sample
- Dashboard:
  - `Message Age P95` rises
  - device may later fall out of `Active Devices (5m)` and appear under `Stale Devices (5m no data)` if no fresh messages arrive

### Sequence Gap

Publish a jump in sequence numbers:

```bash
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/manual-lab-001/telemetry \
  -m "{\"schema_version\":\"1\",\"sensor_id\":\"manual-lab-001\",\"device_id\":\"manual-lab-001\",\"firmware_version\":\"learning-lab-0.1.0\",\"device_state\":\"running\",\"sequence_number\":20,\"telemetry_topic\":\"devices/manual-lab-001/telemetry\",\"temperature\":40.7,\"voltage\":3.27,\"current\":0.10,\"power_usage\":0.34,\"rssi\":-60.0,\"snr\":8.8,\"packet_loss_percent\":0.3,\"free_heap\":180000,\"loop_time_ms\":6.0,\"uptime_seconds\":300,\"reboot_reason\":\"power_on\",\"timestamp\":\"$NOW\"}"
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/manual-lab-001/telemetry \
  -m "{\"schema_version\":\"1\",\"sensor_id\":\"manual-lab-001\",\"device_id\":\"manual-lab-001\",\"firmware_version\":\"learning-lab-0.1.0\",\"device_state\":\"running\",\"sequence_number\":24,\"telemetry_topic\":\"devices/manual-lab-001/telemetry\",\"temperature\":40.9,\"voltage\":3.28,\"current\":0.10,\"power_usage\":0.35,\"rssi\":-59.0,\"snr\":8.9,\"packet_loss_percent\":0.2,\"free_heap\":179500,\"loop_time_ms\":6.2,\"uptime_seconds\":305,\"reboot_reason\":\"power_on\",\"timestamp\":\"$NOW\"}"
```

Expected outcome:

- Log: `mqtt_sequence_gap_detected`
- Metrics: `hardware_telemetry_sequence_gap_total` increments by the detected gap
- Dashboard: `Sequence And Payload Health` shows sequence-gap activity

### Reboot Detection

Publish a lower sequence with reboot evidence:

```bash
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/manual-lab-001/telemetry \
  -m "{\"schema_version\":\"1\",\"sensor_id\":\"manual-lab-001\",\"device_id\":\"manual-lab-001\",\"firmware_version\":\"learning-lab-0.1.0\",\"device_state\":\"running\",\"sequence_number\":40,\"telemetry_topic\":\"devices/manual-lab-001/telemetry\",\"temperature\":41.3,\"voltage\":3.29,\"current\":0.11,\"power_usage\":0.36,\"rssi\":-57.0,\"snr\":9.2,\"packet_loss_percent\":0.1,\"free_heap\":182500,\"loop_time_ms\":6.2,\"uptime_seconds\":800,\"reboot_reason\":\"power_on\",\"timestamp\":\"$NOW\"}"
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/manual-lab-001/telemetry \
  -m "{\"schema_version\":\"1\",\"sensor_id\":\"manual-lab-001\",\"device_id\":\"manual-lab-001\",\"firmware_version\":\"learning-lab-0.1.0\",\"device_state\":\"rebooting\",\"sequence_number\":1,\"telemetry_topic\":\"devices/manual-lab-001/telemetry\",\"temperature\":38.0,\"voltage\":2.95,\"current\":0.05,\"power_usage\":0.15,\"rssi\":-64.0,\"snr\":6.0,\"packet_loss_percent\":1.2,\"free_heap\":175000,\"loop_time_ms\":9.0,\"uptime_seconds\":5,\"reboot_reason\":\"brownout\",\"timestamp\":\"$NOW\"}"
```

Expected outcome:

- Log: `device_reboot_detected`
- Metrics:
  - `hardware_reboot_total` increments
  - `hardware_runtime_uptime_seconds_*` reflects the reset
- Dashboard:
  - `Reboots By Reason` shows `brownout`
  - `Lifecycle State Activity` includes `rebooting`

## Chaos Command Validation

The simulator supports both the legacy command topic and the target per-device command topic.

### Legacy Topic

Send a command to the current default topic:

```bash
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t sensors/sensor-fleet-0/chaos \
  -m '{"command":"signal_loss","duration":"20s","intensity":"high"}'
```

### Per-Device Topic

Send a command to the target topic:

```bash
kubectl run -i --rm mqtt-toolbox \
  --image=eclipse-mosquitto:2 \
  --restart=Never \
  -- mosquitto_pub -h emqx.observability.svc.cluster.local -p 1883 \
  -t devices/sensor-fleet-0/commands \
  -m '{"command":"sleep_mode","duration":"20s","intensity":"medium"}'
```

Supported commands:

- `spike`
- `signal_loss`
- `slow_loop`
- `sleep_mode`
- `malformed_payload`
- `sequence_gap`
- `brownout`
- `memory_leak`

Expected outcome after a chaos command:

- Sensor logs show the command receipt and mode transition
- Ingestor logs show the resulting telemetry effects
- Dashboard behavior changes according to the command:
  - `signal_loss`: worse `Radio Quality` and higher `Packet Loss`
  - `sleep_mode`: lower activity and `sleeping` lifecycle state
  - `slow_loop`: higher `Voltage, Free Heap, And Loop Time` loop-time trace
  - `malformed_payload`: higher invalid payload count
  - `sequence_gap`: higher sequence-gap count
  - `brownout` or `memory_leak`: reboot activity and reboot-reason breakdown

## Useful Queries During Validation

Ingestor logs:

```bash
kubectl logs -n hardware-sim deploy/mqtt-ingestor --tail=100
kubectl logs -n dev-hardware-sim deploy/mqtt-ingestor --tail=100
```

Sensor logs:

```bash
kubectl logs -n hardware-sim statefulset/sensor-fleet --all-pods --tail=100
kubectl logs -n dev-hardware-sim statefulset/sensor-fleet --all-pods --tail=100
```

Dashboard targets to review:

- `Edge Simulation: Sensor Health`
- `Active Devices (5m)`
- `Stale Devices (5m no data)`
- `Lifecycle State Activity`
- `Sequence And Payload Health`
- `Reboots By Reason`
- `Radio Quality`
- `Packet Loss`
- `Voltage, Free Heap, And Loop Time`

## Notes

- This runbook does not require a database write path. The validation target is the observability flow into logs, metrics, and Grafana.
- `docs/architecture/ownership.md` does not need an update for PR 10 because this runbook documents the existing diagnostic path rather than changing ownership boundaries.
