#!/bin/bash
SERVICE_NAME="proxy.service"
CHECK_INTERVAL_MINS="5"
RETRY_MISSING_MINS="5"

SLEEP_SECONDS=$(( CHECK_INTERVAL_MINS * 60 ))
RETRY_SECONDS=$(( RETRY_MISSING_MINS * 60 ))

log() {
    local level=$1
    local msg=$2
    jq -n -c \
        --arg service "tailscale-gate" \
        --arg level "$level" \
        --arg msg "$msg" \
        '{service: $service, level: $level, msg: $msg}'
}

log "INFO" "Service started. Monitoring $SERVICE_NAME."

while true; do
    # 1. RUNNING STATUS CHECK
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        # Ensure Serve is set (Internal)
        tailscale serve --bg --https=8443 http://localhost:8085 > /dev/null 2>&1
        # Ensure Funnel is set (Public) - using the confirmed working syntax
        tailscale funnel --bg --https=8443 8085 > /dev/null 2>&1
        log "INFO" "Proxy RUNNING - Funnel OPEN"
    else
        tailscale funnel --bg --https=8443 off > /dev/null 2>&1
        tailscale serve reset > /dev/null 2>&1
        log "WARN" "Proxy DOWN - Funnel CLOSED"
    fi

    # 2. STANDARD BREATHER
    sleep "$SLEEP_SECONDS"
done