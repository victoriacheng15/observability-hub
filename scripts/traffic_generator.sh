#!/bin/bash
PROXY_URL="http://localhost:8085"
REGIONS=("us-east-1" "us-west-2" "ca-central-1" "eu-west-1" "eu-central-1" "uk-south-1" "asia-east-1" "asia-southeast-1" "asia-south-1")
TIMEZONES=("America/Edmonton" "America/Vancouver" "America/Toronto" "Europe/Dublin" "Europe/Frankfurt" "Europe/London" "Asia/Taipei" "Asia/Singapore" "Asia/Kolkata")
DEVICES=("iphone" "android" "browser" "sensor-node")
NETWORKS=("wifi" "5g" "4g" "ethernet")

# Log helper using jq for safe JSON generation
log() {
    local level=$1
    local msg=$2
    local json_payload
    
    json_payload=$(jq -n -c \
        --arg service "traffic-generator" \
        --arg level "$level" \
        --arg msg "$msg" \
        '{service: $service, level: $level, msg: $msg}')

    echo "$json_payload"

    if command -v logger >/dev/null 2>&1; then
        logger -t "traffic-generator" "$json_payload" || true
    fi
}

generate_cycle() {
  local mode=$1
  local include_health=${2:-true}
  local r_idx=$(( RANDOM % ${#REGIONS[@]} ))
  local d_idx=$(( RANDOM % ${#DEVICES[@]} ))
  local n_idx=$(( RANDOM % ${#NETWORKS[@]} ))
  local hex_id=$(openssl rand -hex 4)

  if [ "$include_health" = "true" ]; then
    curl -s -o /dev/null "$PROXY_URL/api/health"
  fi

  local should_fail=false
  if [ "$mode" = "burst" ] && [ -n "$BURST_FAIL_IDS" ]; then
    for id in $BURST_FAIL_IDS; do
      if [ "$id" -eq "$BURST_CYCLE_COUNT" ]; then
        should_fail=true
        break
      fi
    done
  fi

  if [ "$should_fail" = "true" ]; then
    curl -s -X POST "$PROXY_URL/api/trace/synthetic/fail-$hex_id" \
      -H "Content-Type: application/json" \
      -H "X-Traffic-Mode: burst-fail" \
      -d "{\"region\": \"broken-payload" > /dev/null
    log "WARN" "Injected failure for fail-$hex_id (burst-fail)"
  else
    curl -s -X POST "$PROXY_URL/api/trace/synthetic/$hex_id" \
      -H "Content-Type: application/json" \
      -H "X-Traffic-Mode: $mode" \
      -d "{
        \"region\": \"${REGIONS[$r_idx]}\",
        \"timezone\": \"${TIMEZONES[$r_idx]}\",
        \"device\": \"${DEVICES[$d_idx]}\",
        \"network_type\": \"${NETWORKS[$n_idx]}\"
      }" > /dev/null
    log "INFO" "Generated synthetic trace for $hex_id in ${REGIONS[$r_idx]} ($mode)"
  fi
}

case "$1" in
  --continuous)
    count=1
    while true; do
      generate_cycle "continuous" "true"
      echo "✅ Cycle $count complete. Sleeping for 60s..."
      ((count++)); sleep 60
    done ;;
  --burst)
    echo "🚀 Burst mode: Running 20 rapid cycles (Pure Trace Burst)..."
    num_fails=$(( RANDOM % 5 + 1 ))
    BURST_FAIL_IDS=$(shuf -i 1-20 -n $num_fails | xargs)
    echo "⚠️  Injecting $num_fails failures at cycles: $BURST_FAIL_IDS"
    for i in {1..20}; do
      BURST_CYCLE_COUNT=$i
      generate_cycle "burst" "false"
      sleep 0.5
    done ;;
  *) 
    for i in {1..3}; do
      generate_cycle "cron" "true"
      sleep 1
    done ;;
esac
