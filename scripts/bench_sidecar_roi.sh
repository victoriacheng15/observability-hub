#!/bin/bash

# Sidecar ROI Benchmark (Logs & Metrics)
# Usage: ./scripts/bench_sidecar_roi.sh --type [logs|metrics]

PROCESSOR="/usr/local/bin/obs-processor"
TOKEN_RATIO=4 
COST_PER_1M_USD=2.50
USD_TO_CAD=1.40
TEST_DIR="/tmp/obs-bench"

# Default to logs if no type provided
TYPE="logs"
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --type) TYPE="$2"; shift ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# 0. Environment Setup
if [[ -f .env ]]; then
    export $(grep -E "LOKI_URL|PROMETHEUS_URL" .env | xargs)
fi
LOKI_URL=${LOKI_URL:-"http://localhost:30100"}
PROMETHEUS_URL=${PROMETHEUS_URL:-"http://localhost:30090"}

if [[ ! -f "$PROCESSOR" ]]; then
    echo "Error: Processor binary not found at $PROCESSOR. Run 'make mcp-build' first."
    exit 1
fi

mkdir -p "$TEST_DIR"

# 1. Fetch Phase
fetch_baselines() {
    local start_time
    start_time=$(date -d "24 hours ago" +%s)
    
    echo "--- Phase 1: Fetching 24h Baselines (${TYPE^^}) ---"
    
    if [[ "$TYPE" == "logs" ]]; then
        local services=("mcp" "proxy" "worker.analytics" "worker.ingestion")
        for svc in "${services[@]}"; do
            echo "Fetching logs for $svc..."
            curl -G -s "$LOKI_URL/loki/api/v1/query_range" \
              --data-urlencode "query={service_name=\"$svc\"}" \
              --data-urlencode "limit=1000" \
              --data-urlencode "start=${start_time}000000000" > "$TEST_DIR/${svc}.json"
        done
    else
        local queries=("kepler_node_cpu_joules_total" "node_cpu_seconds_total" "proxy_synthetic_request_total")
        for q in "${queries[@]}"; do
            echo "Fetching metrics for $q..."
            curl -G -s "$PROMETHEUS_URL/api/v1/query_range" \
              --data-urlencode "query=$q" \
              --data-urlencode "start=$start_time" \
              --data-urlencode "end=$(date +%s)" \
              --data-urlencode "step=1m" > "$TEST_DIR/${q}.json"
        done
    fi
    echo "Fetch complete."
    echo ""
}

# 2. Benchmark Phase
TOTAL_RAW_BYTES=0
TOTAL_SUMMARY_BYTES=0
TOTAL_RAW_TOKENS=0
TOTAL_SUMMARY_TOKENS=0

process_file() {
    local file=$1
    [[ -s "$file" ]] || return
    local name=$(basename "$file" ".json")

    local raw_size=$(stat -c %s "$file")
    local processed_json=$(cat "$file" | "$PROCESSOR" --type "$TYPE" 2>/dev/null)
    if [[ $? -ne 0 ]]; then return; fi

    local summary_size=$(echo "$processed_json" | wc -c)
    local raw_tokens=$((raw_size / TOKEN_RATIO))
    local summary_tokens=$((summary_size / TOKEN_RATIO))
    
    local raw_usd=$(echo "scale=4; ($raw_tokens / 1000000) * $COST_PER_1M_USD" | bc)
    local summary_usd=$(echo "scale=4; ($summary_tokens / 1000000) * $COST_PER_1M_USD" | bc)
    local saved_usd=$(echo "scale=4; $raw_usd - $summary_usd" | bc)
    local saved_cad=$(echo "scale=4; $saved_usd * $USD_TO_CAD" | bc)

    # Output Row
    printf "%-25s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
        "${name:0:25}" "$raw_size" "$raw_tokens" "$summary_size" "$summary_tokens" "$saved_usd" "$saved_cad"

    TOTAL_RAW_BYTES=$((TOTAL_RAW_BYTES + raw_size))
    TOTAL_SUMMARY_BYTES=$((TOTAL_SUMMARY_BYTES + summary_size))
    TOTAL_RAW_TOKENS=$((TOTAL_RAW_TOKENS + raw_tokens))
    TOTAL_SUMMARY_TOKENS=$((TOTAL_SUMMARY_TOKENS + summary_tokens))
}

fetch_baselines

echo "--- Phase 2: Benchmarking Suite: ${TYPE^^} (24 hours) ---"
# Print Table Header - 25 | 23 | 23 | 23
printf "%-25s | %-23s | %-23s | %-23s\n" " " "BEFORE" "AFTER (RUST)" "SAVINGS"
printf "%-25s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
    "Service" "Bytes" "Tokens" "Bytes" "Tokens" "USD" "CAD"
printf "%-25s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
    "-------------------------" "----------" "----------" "----------" "----------" "----------" "----------"

for f in "$TEST_DIR"/*.json; do
    process_file "$f"
done

# Grand Totals
TOTAL_SAVED_TOKENS=$((TOTAL_RAW_TOKENS - TOTAL_SUMMARY_TOKENS))
TOTAL_RAW_USD=$(echo "scale=4; ($TOTAL_RAW_TOKENS / 1000000) * $COST_PER_1M_USD" | bc)
TOTAL_SUMMARY_USD=$(echo "scale=4; ($TOTAL_SUMMARY_TOKENS / 1000000) * $COST_PER_1M_USD" | bc)
TOTAL_SAVED_USD=$(echo "scale=4; $TOTAL_RAW_USD - $TOTAL_SUMMARY_USD" | bc)
TOTAL_SAVED_CAD=$(echo "scale=4; $TOTAL_SAVED_USD * $USD_TO_CAD" | bc)

printf "%-25s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
    "-------------------------" "----------" "----------" "----------" "----------" "----------" "----------"
printf "%-25s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
    "AGGREGATE" "$TOTAL_RAW_BYTES" "$TOTAL_RAW_TOKENS" "$TOTAL_SUMMARY_BYTES" "$TOTAL_SUMMARY_TOKENS" "$TOTAL_SAVED_USD" "$TOTAL_SAVED_CAD"

echo ""
echo "========================================="
echo "         FINOPS ROI SUMMARY (${TYPE^^})"
echo "========================================="
echo "Total Tokens Saved:   $TOTAL_SAVED_TOKENS"
echo "Total USD Savings:    \$${TOTAL_SAVED_USD}"
echo "Total CAD Savings:    \$${TOTAL_SAVED_CAD}"
if [[ $TOTAL_SUMMARY_BYTES -gt 0 ]]; then
    echo "Context Gain:         ~$(echo "scale=1; $TOTAL_RAW_BYTES / $TOTAL_SUMMARY_BYTES" | bc)x more dense"
fi
echo "========================================="

# 3. Cleanup
rm -rf "$TEST_DIR"
