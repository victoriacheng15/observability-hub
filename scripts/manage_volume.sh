#!/usr/bin/env bash
set -euo pipefail

# --- CONFIG ---
BACKUP_BASE="/home/server/backups"
RETENTION_DAYS=7

# Embedded volume list as a clean Bash array
VOLUMES=(
  postgres_data
  grafana_data
  loki_data
  promtail_data
)

# --- LOGGING ---
# Log helper for structured JSON output
log() {
    local level=$1
    local msg=$2
    jq -n -c \
        --arg service "volume-manager" \
        --arg level "$level" \
        --arg msg "$msg" \
        '{service: $service, level: $level, msg: $msg}'
}

error() {
    log "ERROR" "$1"
    exit 1
}

# --- VALIDATION ---
validate_docker() {
  if ! command -v docker >/dev/null; then
    error "Docker is not installed or not in PATH"
  fi
  if ! docker info &>/dev/null; then
    error "Docker is not running or not accessible"
  fi
}

# --- CREATE VOLUMES FUNCTION ---
create_volumes() {
  validate_docker

  for vol in "${VOLUMES[@]}"; do
    [[ -z "$vol" ]] && continue

    if ! docker volume inspect "$vol" &>/dev/null; then
      docker volume create "$vol" >/dev/null
      log "INFO" "Created volume: '$vol'"
    fi
  done
}

# --- BACKUP VOLUMES FUNCTION ---
backup_volumes() {
  validate_docker

  # Stop services for consistency
  docker compose stop

  DATE=$(date +%Y-%m-%d)
  BACKUP_DIR="$BACKUP_BASE/$DATE"
  mkdir -p "$BACKUP_DIR"

  log "INFO" "Creating backup in $BACKUP_DIR for volumes: ${VOLUMES[*]}"

  for vol in "${VOLUMES[@]}"; do
    [[ -z "$vol" ]] && continue

    if ! docker volume inspect "$vol" &>/dev/null; then
      log "WARN" "Volume '$vol' does not exist. Skipping."
      continue
    fi

    # Use container to backup (no sudo, no path assumptions)
    docker run --rm \
      -v "$vol":/volume \
      -v "$BACKUP_DIR":/backup \
      alpine tar -czf "/backup/${vol}.tar.gz" -C /volume .
  done

  # Restart services
  docker compose start

  # Cleanup old backups
  find "$BACKUP_BASE" -maxdepth 1 -type d -name "20[0-9][0-9]-[0-1][0-9]-[0-3][0-9]" -mtime +$RETENTION_DAYS -exec rm -rf {} + 2>/dev/null || true

  log "INFO" "Backup job completed successfully"
}

# --- RESTORE VOLUMES FUNCTION ---
restore_volumes() {
  validate_docker

  BACKUP_DIR=$(ls -1dt "$BACKUP_BASE"/20[0-9][0-9]-[0-1][0-9]-[0-3][0-9]/ 2>/dev/null | head -n1)
  if [[ -z "$BACKUP_DIR" ]]; then
    error "No dated backup folders found in: $BACKUP_BASE"
  fi
  BACKUP_DIR="${BACKUP_DIR%/}"
  log "INFO" "Restoring volumes from backup: $(basename "$BACKUP_DIR")"

  if ! docker compose down; then
    log "WARN" "Some services may not have stopped cleanly."
  fi

  for vol in "${VOLUMES[@]}"; do
    [[ -z "$vol" ]] && continue

    BACKUP_FILE="$BACKUP_DIR/${vol}.tar.gz"
    if [[ ! -f "$BACKUP_FILE" ]]; then
      log "WARN" "Skipping $vol: backup file not found ($BACKUP_FILE)"
      continue
    fi

    if docker volume inspect "$vol" &>/dev/null; then
      docker volume rm -f "$vol" >/dev/null
    fi

    docker volume create "$vol" >/dev/null
    docker run --rm \
      -v "$vol":/restore-target \
      -v "$BACKUP_DIR":/backup \
      alpine sh -c "cd /restore-target && tar xzf /backup/$(basename "$BACKUP_FILE") --no-same-owner && \
        case \"$vol\" in \
          jenkins_data) chown -R 1000:1000 . ;; \
          gitea_data) chown -R 1000:1000 . ;; \
          grafana_data) chown -R 472:472 . ;; \
          postgres_data) chown -R 999:999 . ;; \
          *) echo \"No ownership fix needed for $vol\" ;; \
        esac"
  done

  if docker compose up -d; then
    log "INFO" "Restore completed and services restarted successfully!"
  else
    error "Failed to restart services. Check 'docker compose logs'."
  fi
}

# --- MAIN ---
case "${1:-}" in
  create|setup)
    create_volumes
    ;;
  backup)
    backup_volumes
    ;;
  restore)
    restore_volumes
    ;;
  *)
    echo "Usage: $0 {create|backup|restore}" >&2
    exit 1
    ;;
esac