#!/bin/bash

# Log Compression Benchmark with Granular FinOps Comparison Table
# Usage: ./scripts/bench_log_compression.sh [file_or_dir]

TEST_DIR="internal/mcp/tools/log-processor/tests"
PROCESSOR="./internal/mcp/tools/log-processor/target/release/log-processor"
TOKEN_RATIO=4 
COST_PER_1M_USD=2.50
USD_TO_CAD=1.40

# 0. Environment Setup
if [[ -f .env ]]; then
    export $(grep LOKI_URL .env | xargs)
fi
LOKI_URL=${LOKI_URL:-"http://localhost:30100"}

if [[ ! -f "$PROCESSOR" ]]; then
    echo "Error: Processor binary not found. Run 'cargo build --release' first."
    exit 1
fi

# 1. Fetch Phase
fetch_baselines() {
    local services=("mcp" "proxy" "worker.analytics" "worker.ingestion")
    local start_time
    start_time=$(date -d "24 hours ago" +%s)000000000
    echo "--- Phase 1: Fetching 24h Baselines ---"
    mkdir -p "$TEST_DIR"
    for svc in "${services[@]}"; do
        echo "Fetching $svc..."
        curl -G -s "$LOKI_URL/loki/api/v1/query_range" \
          --data-urlencode "query={service_name=\"$svc\"}" \
          --data-urlencode "limit=1000" \
          --data-urlencode "start=$start_time" > "$TEST_DIR/${svc}_24h.json"
    done
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
    local svc_name=$(basename "$file" "_24h.json")

    local raw_size=$(stat -c %s "$file")
    local processed_json=$(cat "$file" | "$PROCESSOR" 2>/dev/null)
    if [[ $? -ne 0 ]]; then return; fi

    local summary_size=$(echo "$processed_json" | wc -c)
    local raw_tokens=$((raw_size / TOKEN_RATIO))
    local summary_tokens=$((summary_size / TOKEN_RATIO))
    
    local raw_usd=$(echo "scale=4; ($raw_tokens / 1000000) * $COST_PER_1M_USD" | bc)
    local summary_usd=$(echo "scale=4; ($summary_tokens / 1000000) * $COST_PER_1M_USD" | bc)
    local saved_usd=$(echo "scale=4; $raw_usd - $summary_usd" | bc)
    local saved_cad=$(echo "scale=4; $saved_usd * $USD_TO_CAD" | bc)

    # Output Row with separate columns
    printf "%-18s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
        "$svc_name" "$raw_size" "$raw_tokens" "$summary_size" "$summary_tokens" "$saved_usd" "$saved_cad"

    TOTAL_RAW_BYTES=$((TOTAL_RAW_BYTES + raw_size))
    TOTAL_SUMMARY_BYTES=$((TOTAL_SUMMARY_BYTES + summary_size))
    TOTAL_RAW_TOKENS=$((TOTAL_RAW_TOKENS + raw_tokens))
    TOTAL_SUMMARY_TOKENS=$((TOTAL_SUMMARY_TOKENS + summary_tokens))
}

# Execute Data Retrieval
fetch_baselines

# Execute Benchmarking
echo "--- Phase 2: Benchmarking Suite ---"
TARGET=${1:-"$TEST_DIR"}

# Print Table Header with separate columns
printf "%-18s | %-23s | %-23s | %-23s\n" " " "BEFORE (RAW)" "AFTER (RUST)" "SAVINGS"
printf "%-18s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
    "Service" "Bytes" "Tokens" "Bytes" "Tokens" "USD" "CAD"
printf "%-18s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
    "------------------" "----------" "----------" "----------" "----------" "----------" "----------"

if [[ -f "$TARGET" ]]; then
    process_file "$TARGET"
elif [[ -d "$TARGET" ]]; then
    for f in "$TARGET"/*.json; do
        [[ -e "$f" ]] || continue
        if [[ "$(basename "$f")" != "sample_logs.json" && "$(basename "$f")" != "log_storm.json" ]]; then
             process_file "$f"
        fi
    done
fi

# Grand Totals
TOTAL_SAVED_TOKENS=$((TOTAL_RAW_TOKENS - TOTAL_SUMMARY_TOKENS))
TOTAL_RAW_USD=$(echo "scale=4; ($TOTAL_RAW_TOKENS / 1000000) * $COST_PER_1M_USD" | bc)
TOTAL_SUMMARY_USD=$(echo "scale=4; ($TOTAL_SUMMARY_TOKENS / 1000000) * $COST_PER_1M_USD" | bc)
TOTAL_SAVED_USD=$(echo "scale=4; $TOTAL_RAW_USD - $TOTAL_SUMMARY_USD" | bc)
TOTAL_SAVED_CAD=$(echo "scale=4; $TOTAL_SAVED_USD * $USD_TO_CAD" | bc)

printf "%-18s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
    "------------------" "----------" "----------" "----------" "----------" "----------" "----------"
printf "%-18s | %-10s | %-10s | %-10s | %-10s | %-10s | %-10s\n" \
    "AGGREGATE" "$TOTAL_RAW_BYTES" "$TOTAL_RAW_TOKENS" "$TOTAL_SUMMARY_BYTES" "$TOTAL_SUMMARY_TOKENS" "$TOTAL_SAVED_USD" "$TOTAL_SAVED_CAD"

echo ""
echo "========================================="
echo "         FINOPS ROI SUMMARY"
echo "========================================="
echo "Total Tokens Saved:   $TOTAL_SAVED_TOKENS"
echo "Total USD Savings:    \$${TOTAL_SAVED_USD}"
echo "Total CAD Savings:    \$${TOTAL_SAVED_CAD}"
echo "Context Gain:         ~$(echo "scale=1; $TOTAL_RAW_BYTES / $TOTAL_SUMMARY_BYTES" | bc)x more dense"
echo "========================================="

# 3. Cleanup
echo "--- Phase 3: Cleanup ---"
for f in "$TEST_DIR"/*_24h.json; do
    [[ -f "$f" ]] && rm "$f"
done
