#!/bin/bash
PROXY_URL="http://localhost:8085"
REGIONS=("us-east-1" "us-west-2" "ca-central-1" "eu-west-1" "eu-central-1" "uk-south-1" "asia-east-1" "asia-southeast-1" "asia-south-1")
TIMEZONES=("America/Edmonton" "America/Vancouver" "America/Toronto" "Europe/Dublin" "Europe/Frankfurt" "Europe/London" "Asia/Taipei" "Asia/Singapore" "Asia/Kolkata")
DEVICES=("iphone" "android" "browser" "sensor-node")
NETWORKS=("wifi" "5g" "4g" "ethernet")

# Log helper using jq for safe JSON generation
# Ensures newlines and quotes in 'msg' are properly escaped
log() {
    local level=$1
    local msg=$2
    local json_payload
    
    # Generate JSON payload
    json_payload=$(jq -n -c \
        --arg service "traffic-generator" \
        --arg level "$level" \
        --arg msg "$msg" \
        '{service: $service, level: $level, msg: $msg}')

    # 1. Output to stdout
    echo "$json_payload"

    # 2. Send directly to system journal
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

  # Baseline health check (optional)
  if [ "$include_health" = "true" ]; then
    curl -s -o /dev/null "$PROXY_URL/api/health"
  fi

  # Synthetic Trace with high-signal metadata
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
    local_health="true"
    if [ "$2" = "--no-health" ]; then
      local_health="false"
      echo "🚀 Pure Burst mode (No health checks)..."
    else
      echo "🚀 Burst mode: Running 20 rapid cycles (with 2 health checks)..."
    fi

    for i in {1..20}; do
      if [ "$local_health" = "true" ] && [ $i -le 2 ]; then
        generate_cycle "burst" "true"
      else
        generate_cycle "burst" "false"
      fi
      sleep 0.5
    done ;;
  *) generate_cycle "cron" "true" ;;
esac
