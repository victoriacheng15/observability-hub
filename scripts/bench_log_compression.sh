#!/bin/bash

# Log Compression Benchmark with Automated 24h Fetching and Cleanup
# Usage: ./scripts/bench_log_compression.sh [file_or_dir]

TEST_DIR="internal/mcp/tools/log-processor/tests"
PROCESSOR="./internal/mcp/tools/log-processor/target/release/log-processor"
TOKEN_RATIO=4 # 1 token ≈ 4 characters

# 0. Environment Setup
if [[ -f .env ]]; then
    export $(grep LOKI_URL .env | xargs)
fi

LOKI_URL=${LOKI_URL:-"http://localhost:30100"}

if [[ ! -f "$PROCESSOR" ]]; then
    echo "Error: Processor binary not found. Run 'cargo build --release' first."
    exit 1
fi

# 1. Fetch Phase: Retrieve 24h baselines for the 4 core services
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
TOTAL_RAW_LINES=0
TOTAL_SUMMARY_LINES=0

process_file() {
    local file=$1
    [[ -s "$file" ]] || return # Skip empty files

    echo "--- Benchmarking: $(basename "$file") ---"

    local raw_size
    raw_size=$(stat -c %s "$file")
    
    local processed_json
    processed_json=$(cat "$file" | "$PROCESSOR" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to process $file"
        return
    fi

    local summary_size
    summary_size=$(echo "$processed_json" | wc -c)

    local raw_lines
    raw_lines=$(echo "$processed_json" | grep -oP '"total_raw_lines":\s*\K\d+')
    local summarized_lines
    summarized_lines=$(echo "$processed_json" | grep -oP '"summarized_count":\s*\K\d+')

    if [[ -z "$raw_lines" || "$raw_lines" -eq 0 ]]; then
        echo "No log lines found."
        echo "---------------------------------"
        return
    fi

    local raw_tokens=$((raw_size / TOKEN_RATIO))
    local summary_tokens=$((summary_size / TOKEN_RATIO))
    local tokens_saved=$((raw_tokens - summary_tokens))
    local reduction
    reduction=$(echo "scale=2; (1 - $summary_size / $raw_size) * 100" | bc)

    echo "Size:      $raw_size -> $summary_size bytes ($reduction% reduction)"
    echo "Tokens:    $raw_tokens -> $summary_tokens (~$tokens_saved saved)"
    echo "Lines:     $raw_lines -> $summarized_lines"
    echo "---------------------------------"

    # Accumulate totals
    TOTAL_RAW_BYTES=$((TOTAL_RAW_BYTES + raw_size))
    TOTAL_SUMMARY_BYTES=$((TOTAL_SUMMARY_BYTES + summary_size))
    TOTAL_RAW_LINES=$((TOTAL_RAW_LINES + raw_lines))
    TOTAL_SUMMARY_LINES=$((TOTAL_SUMMARY_LINES + summarized_lines))
}

# Execute Fetching
fetch_baselines

# Execute Benchmarking
echo "--- Phase 2: Benchmarking Suite ---"
TARGET=${1:-"$TEST_DIR"}

if [[ -f "$TARGET" ]]; then
    process_file "$TARGET"
elif [[ -d "$TARGET" ]]; then
    for f in "$TARGET"/*.json; do
        [[ -e "$f" ]] || continue
        process_file "$f"
    done
fi

# Final Aggregate Summary
if [[ $TOTAL_RAW_BYTES -gt 0 ]]; then
    TOTAL_RAW_TOKENS=$((TOTAL_RAW_BYTES / TOKEN_RATIO))
    TOTAL_SUMMARY_TOKENS=$((TOTAL_SUMMARY_BYTES / TOKEN_RATIO))
    GRAND_SAVINGS=$((TOTAL_RAW_TOKENS - TOTAL_SUMMARY_TOKENS))
    TOTAL_REDUCTION=$(echo "scale=2; (1 - $TOTAL_SUMMARY_BYTES / $TOTAL_RAW_BYTES) * 100" | bc)

    echo "========================================="
    echo "         AGGREGATE BENCHMARK SUMMARY"
    echo "========================================="
    echo "Total Raw Size:      $TOTAL_RAW_BYTES bytes (~$TOTAL_RAW_TOKENS tokens)"
    echo "Total Summary Size:  $TOTAL_SUMMARY_BYTES bytes (~$TOTAL_SUMMARY_TOKENS tokens)"
    echo "Total Byte Reduction: $TOTAL_REDUCTION%"
    echo "Total Tokens Saved:   $GRAND_SAVINGS"
    echo "Total Lines:         $TOTAL_RAW_LINES -> $TOTAL_SUMMARY_LINES"
    echo "========================================="
fi

# 3. Cleanup Phase: Remove the fetched 24h files
echo "--- Phase 3: Cleanup ---"
for f in "$TEST_DIR"/*_24h.json; do
    if [[ -f "$f" ]]; then
        rm "$f"
        echo "Removed $(basename "$f")"
    fi
done
echo "Cleanup complete."
