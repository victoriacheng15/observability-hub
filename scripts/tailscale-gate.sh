#!/bin/bash
CONTAINER="proxy_server"
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

log "INFO" "Service started. Monitoring $CONTAINER."

while true; do
    # 1. EXISTENCE CHECK
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER}$" ; then
        log "WARN" "Container $CONTAINER not found. Retrying in ${RETRY_MISSING_MINS}m."
        sleep "$RETRY_SECONDS"
        continue
    fi

    # 2. RUNNING STATUS CHECK
    IS_RUNNING=$(docker inspect -f '{{.State.Running}}' $CONTAINER 2>/dev/null)

    if [ "$IS_RUNNING" == "true" ]; then
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

    # 3. STANDARD BREATHER
    sleep "$SLEEP_SECONDS"
done