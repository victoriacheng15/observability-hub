#!/bin/bash
set -euo pipefail

# OTel-aligned service variables
SERVICE_NAME="gitops.sync"
JOB_NAME="bash.automation"
REPO_NAME=${1:-"observability-hub"} # Default to observability-hub as per original intent

BASE_DIR="/home/server/software"

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

    # Add optional 'repo' if it exists (for gitops.sync)
    if [[ -n "$REPO_NAME" ]]; then
        json_args+=(--arg repo "$REPO_NAME")
        query+=' + {repo: $repo}'
    fi

    # Generate the final JSON payload
    local json_payload
    json_payload=$(jq -n -c "${json_args[@]}" "$query")

    # 1. Output to stdout (captured by Go parent process)
    echo "$json_payload"

    # 2. Send directly to system journal (viewable via `journalctl -t gitops.sync`)
    if command -v logger >/dev/null 2>&1; then
        logger -t "$SERVICE_NAME" "$json_payload" || true
    fi
}

# 1. Validation Logic (Security Barrier)
if [[ -z "$REPO_NAME" ]]; then
    log "ERROR" "No repository name provided."
    exit 1
fi

REPO_PATH="${BASE_DIR}/${REPO_NAME}"

if [[ ! -d "$REPO_PATH/.git" ]]; then
    log "ERROR" "Repository '${REPO_NAME}' is not a valid git repository."
    exit 1
fi

# 1b. Opt-in Check
# The presence of a '.gitops' file in the repo root is required this sync process.
if [[ ! -f "$REPO_PATH/.gitops" ]]; then
    log "CRITICAL" "Repository '${REPO_NAME}' lacks '.gitops' marker. Access denied."
    exit 1
fi

# 2. Sync Logic
cd "$REPO_PATH"

TARGET_BRANCH="main"
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

if [[ "$CURRENT_BRANCH" != "$TARGET_BRANCH" ]]; then
    # Check for uncommitted changes to TRACKED files (Dirty State)
    # Ignores untracked files so scratch files don't block the sync
    if [[ -n $(git status --untracked-files=no --porcelain) ]]; then
        log "WARN" "Uncommitted changes detected on ($CURRENT_BRANCH). Skipping sync to prevent data loss."
        exit 0
    fi

    # If the tree is clean, we can safely switch to main regardless of PR state
    if ! git checkout "$TARGET_BRANCH" >/dev/null 2>&1; then
        log "ERROR" "Failed to switch to $TARGET_BRANCH. Check permissions."
        exit 1
    fi
fi

git branch --set-upstream-to=origin/main main >/dev/null 2>&1 || true

# Atomic Fetch
if ! git fetch origin "$TARGET_BRANCH" --quiet; then
    log "ERROR" "Failed to fetch from origin. Check network/permissions."
    exit 1
fi

LOCAL_HASH=$(git rev-parse HEAD)
REMOTE_HASH=$(git rev-parse "origin/$TARGET_BRANCH")

if [[ "$LOCAL_HASH" != "$REMOTE_HASH" ]]; then
    # Atomic Sync: Transition from git pull to git fetch + git merge --ff-only
    # to prevent accidental merge commits and ensure clean fast-forwards.
    if OUTPUT=$(git merge --ff-only "origin/$TARGET_BRANCH" 2>&1); then
        SAFE_OUTPUT=$(echo "$OUTPUT" | head -c 2048)
        if [[ ${#OUTPUT} -gt 2048 ]]; then SAFE_OUTPUT="${SAFE_OUTPUT}... (truncated)"; fi
        log "INFO" "Sync successful: $SAFE_OUTPUT"
    else
        SAFE_OUTPUT=$(echo "$OUTPUT" | head -c 2048)
        log "ERROR" "Merge failed (non-fast-forward?): $SAFE_OUTPUT"
        exit 1
    fi
fi

# 3. Cleanup Logic (delete all local branches except main)
LOCAL_BRANCHES=$(git branch --format='%(refname:short)' 2>/dev/null | grep -v '^main$' || true)
if [[ -n "$LOCAL_BRANCHES" ]]; then
    # Capture output of branch deletion
    if OUTPUT=$(echo "$LOCAL_BRANCHES" | xargs -r git branch -D 2>&1); then
        SAFE_OUTPUT=$(echo "$OUTPUT" | head -c 2048)
        if [[ ${#OUTPUT} -gt 2048 ]]; then SAFE_OUTPUT="${SAFE_OUTPUT}... (truncated)"; fi
        log "INFO" "$SAFE_OUTPUT"
    else
        SAFE_OUTPUT=$(echo "$OUTPUT" | head -c 2048)
        log "WARN" "Failed to delete some branches: $SAFE_OUTPUT"
    fi
fi