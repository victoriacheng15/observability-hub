#!/usr/bin/env bash
set -euo pipefail

# Benchmarks Go vs Rust obs processors using the same API payload.
#
# Default usage:
#   ./scripts/benchmark_obs_processor.sh
#
# Example usage:
#   ./scripts/benchmark_obs_processor.sh \
#     --type metrics \
#     --iterations 30

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_FILE="${ROOT_DIR}/bench/obs_processor.go"
RUST_FILE="${ROOT_DIR}/bench/obs_processor.rs"

LOGS_URL="${LOKI_URL:-http://localhost:30100}"
METRICS_URL="${PROMETHEUS_URL:-http://localhost:30090}"
LOGS_QUERY='{service_name="proxy"}'
METRICS_QUERY='up'
ITERATIONS=20
TYPE="logs"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --iterations) ITERATIONS="$2"; shift 2 ;;
    --type) TYPE="$2"; shift 2 ;;
    -h|--help)
      sed -n '1,40p' "$0"
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ "$TYPE" != "logs" && "$TYPE" != "metrics" ]]; then
  echo "--type must be logs or metrics" >&2
  exit 1
fi

if [[ ! -f "$GO_FILE" ]]; then
  echo "Missing file: $GO_FILE" >&2
  exit 1
fi
if [[ ! -f "$RUST_FILE" ]]; then
  echo "Missing file: $RUST_FILE" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi
if ! command -v go >/dev/null 2>&1; then
  echo "go is required" >&2
  exit 1
fi
if ! command -v cargo >/dev/null 2>&1; then
  echo "cargo is required" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

PAYLOAD_FILE="${TMP_DIR}/payload.json"
GO_BIN="${TMP_DIR}/obs-go"
RUST_PROJ="${TMP_DIR}/rust-proj"
RUST_BIN="${RUST_PROJ}/target/release/obs-processor-rust-standalone"

END_NS="$(date +%s)000000000"
START_NS="$(( $(date +%s) - 86400 ))000000000"

if [[ "$TYPE" == "logs" ]]; then
  echo "Fetching Loki logs payload (24h)..."
  curl -fsS -G "${LOGS_URL%/}/loki/api/v1/query_range" \
    --data-urlencode "query=${LOGS_QUERY}" \
    --data-urlencode "start=${START_NS}" \
    --data-urlencode "end=${END_NS}" \
    > "$PAYLOAD_FILE"
else
  echo "Fetching Prometheus metrics payload (24h)..."
  START_S="${START_NS:0:-9}"
  END_S="${END_NS:0:-9}"
  curl -fsS -G "${METRICS_URL%/}/api/v1/query_range" \
    --data-urlencode "query=${METRICS_QUERY}" \
    --data-urlencode "start=${START_S}" \
    --data-urlencode "end=${END_S}" \
    --data-urlencode "step=60s" \
    > "$PAYLOAD_FILE"
fi

if [[ ! -s "$PAYLOAD_FILE" ]]; then
  echo "API payload is empty." >&2
  exit 1
fi

echo "Building Go binary..."
go build -o "$GO_BIN" "$GO_FILE"

echo "Building Rust binary..."
mkdir -p "${RUST_PROJ}/src"
cat > "${RUST_PROJ}/Cargo.toml" <<'EOF'
[package]
name = "obs-processor-rust-standalone"
version = "0.1.0"
edition = "2021"

[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
EOF
cp "$RUST_FILE" "${RUST_PROJ}/src/main.rs"
cargo build --release --manifest-path "${RUST_PROJ}/Cargo.toml" --quiet

run_once_ms() {
  local cmd="$1"
  local start_ns end_ns elapsed_ns
  start_ns="$(date +%s%N)"
  eval "$cmd" < "$PAYLOAD_FILE" > /dev/null
  end_ns="$(date +%s%N)"
  elapsed_ns=$((end_ns - start_ns))
  awk "BEGIN { printf \"%.3f\", ${elapsed_ns}/1000000 }"
}

summarize_times() {
  local label="$1"
  shift
  local times=("$@")

  local min max sum avg
  min="${times[0]}"
  max="${times[0]}"
  sum=0

  for t in "${times[@]}"; do
    awk "BEGIN { if (${t} < ${min}) print \"${t}\"; else print \"${min}\" }" >/tmp/.bench_min.$$
    min="$(cat /tmp/.bench_min.$$)"
    awk "BEGIN { if (${t} > ${max}) print \"${t}\"; else print \"${max}\" }" >/tmp/.bench_max.$$
    max="$(cat /tmp/.bench_max.$$)"
    sum="$(awk "BEGIN { printf \"%.6f\", ${sum} + ${t} }")"
  done

  avg="$(awk "BEGIN { printf \"%.3f\", ${sum} / ${#times[@]} }")"
  rm -f /tmp/.bench_min.$$ /tmp/.bench_max.$$

  printf "%-8s avg=%8sms min=%8sms max=%8sms (n=%s)\n" "$label" "$avg" "$min" "$max" "${#times[@]}"
}

echo "Running benchmark (${ITERATIONS} iterations, type=${TYPE})..."
go_times=()
rust_times=()

# Warm-up
"$GO_BIN" --type "$TYPE" < "$PAYLOAD_FILE" > /dev/null
"$RUST_BIN" --type "$TYPE" < "$PAYLOAD_FILE" > /dev/null

for ((i=1; i<=ITERATIONS; i++)); do
  go_ms="$(run_once_ms "\"$GO_BIN\" --type \"$TYPE\"")"
  rust_ms="$(run_once_ms "\"$RUST_BIN\" --type \"$TYPE\"")"
  go_times+=("$go_ms")
  rust_times+=("$rust_ms")
done

echo
echo "Benchmark result (same API payload):"
summarize_times "Go" "${go_times[@]}"
summarize_times "Rust" "${rust_times[@]}"
