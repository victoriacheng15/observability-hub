#!/bin/bash
set -euo pipefail

REPO_NAME=${1:-""}
ALLOWED_REPOS=("observability-hub")
BASE_DIR="/home/server/software"

# Log helper to make Grafana/Loki filtering easy
log() {
    local level=$1
    local msg=$2
    echo "repo=${REPO_NAME:-unknown} level=${level} msg=\"${msg}\""
}

# 1. Validation Logic (Security Barrier)
if [[ -z "$REPO_NAME" ]]; then
    log "ERROR" "No repository name provided."
    exit 1
fi

IS_ALLOWED=false
for repo in "${ALLOWED_REPOS[@]}"; do
    if [[ "$repo" == "$REPO_NAME" ]]; then
        IS_ALLOWED=true
        break
    fi
done

if [[ "$IS_ALLOWED" == "false" ]]; then
    log "CRITICAL" "Repository not in allowlist. Access denied."
    exit 1
fi

REPO_PATH="${BASE_DIR}/${REPO_NAME}"

if [[ ! -d "$REPO_PATH/.git" ]]; then
    log "ERROR" "Path ${REPO_PATH} is not a valid git repository."
    exit 1
fi

# 2. Sync Logic
cd "$REPO_PATH"

# Safety Barrier: Check for uncommitted changes
if [[ -n $(git status --porcelain) ]]; then
    log "ERROR" "Uncommitted changes detected. Aborting sync to prevent data loss."
    exit 1
fi

TARGET_BRANCH="main"
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

if [[ "$CURRENT_BRANCH" != "$TARGET_BRANCH" ]]; then
    log "WARN" "Current branch ($CURRENT_BRANCH) is not $TARGET_BRANCH. Switching..."
    if ! git checkout "$TARGET_BRANCH"; then
        log "ERROR" "Failed to switch to $TARGET_BRANCH. Check for uncommitted changes."
        exit 1
    fi
fi

if ! git fetch origin "$TARGET_BRANCH" --quiet; then
    log "ERROR" "Failed to fetch from origin. Check network/permissions."
    exit 1
fi

LOCAL_HASH=$(git rev-parse HEAD)
REMOTE_HASH=$(git rev-parse origin/main)

if [[ "$LOCAL_HASH" != "$REMOTE_HASH" ]]; then
    log "SUCCESS" "New changes detected. Synchronizing..."
    git pull origin main
else
    # INFO level can be filtered out in Grafana to reduce noise
    log "INFO" "Already in sync."
fi

# 3. Cleanup Logic (delete all local branches except main)
log "INFO" "Cleaning up local branches..."

LOCAL_BRANCHES=$(git branch --format='%(refname:short)' 2>/dev/null | grep -v '^main$' || true)
if [[ -n "$LOCAL_BRANCHES" ]]; then
    log "INFO" "Local branches to delete: ${LOCAL_BRANCHES}"
    echo "$LOCAL_BRANCHES" | xargs -r git branch -D
    log "SUCCESS" "Deleted local branches: ${LOCAL_BRANCHES}"
else
    log "INFO" "No local branches to delete"
fi
