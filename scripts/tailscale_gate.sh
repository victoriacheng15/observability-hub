#!/bin/bash

# OTel-aligned service variables
SERVICE_NAME="tailscale.gate"
JOB_NAME="bash.automation"

# Script-specific variables
PROXY_SERVICE_NAME="proxy.service"
CHECK_INTERVAL_MINS="5"
RETRY_MISSING_MINS="5"

SLEEP_SECONDS=$(( CHECK_INTERVAL_MINS * 60 ))
RETRY_SECONDS=$(( RETRY_MISSING_MINS * 60 ))

# OTel-aligned structured logging function
log() {
    local level=$1
    local msg=$2
    local hostname
    hostname=$(hostname)
    
    # Base JSON object
    local json_args=(
        --arg service "$SERVICE_NAME"
        --arg job "$JOB_NAME"
        --arg level "$level"
        --arg msg "$msg"
        --arg host "$hostname"
    )
    local query='{service: $service, job: $job, level: $level, msg: $msg, "host.name": $host}'

    # This script does not have a repo, so we don't add it.
    
    # Generate the final JSON payload
    local json_payload
    json_payload=$(jq -n -c "${json_args[@]}" "$query")

    echo "$json_payload"
    # Also send to system journal for out-of-band debugging
    if command -v logger >/dev/null 2>&1; then
        logger -t "$SERVICE_NAME" "$json_payload" || true
    fi
}

log "INFO" "Service started. Monitoring $PROXY_SERVICE_NAME."

while true; do
    # 1. RUNNING STATUS CHECK
    if systemctl is-active --quiet "$PROXY_SERVICE_NAME"; then
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