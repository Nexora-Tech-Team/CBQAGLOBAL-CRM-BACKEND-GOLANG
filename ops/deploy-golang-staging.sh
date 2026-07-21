#!/bin/bash
# Deploys a new erp-cbqa-global binary to the STAGING box for this API
# (212.85.25.165): backup current binary, apply any new migration SQL
# (idempotent, IF NOT EXISTS-guarded), swap the binary, restart the
# systemd service, health-check it, auto-rollback on failure.
#
# Usage: deploy-golang-staging.sh <path-to-extracted-artifact-dir>
#   (that dir must contain ./erp-cbqa-global and optionally ./migrate/*.sql)
#
# Runs on the server itself, invoked over SSH by
# .github/workflows/deploy-staging.yml — mirrors deploy-golang-prod.sh,
# see ops/README.md.

set -euo pipefail

SRC_DIR="${1:?Usage: deploy-golang-staging.sh <extracted-artifact-dir>}"
APP_DIR="/root/app/golang-pm-staging"
BIN_PATH="$APP_DIR/erp-cbqa-global"
BACKUP_DIR="$APP_DIR/backups"
SERVICE_NAME="cbqaglobal-golang-pm-staging"
HEALTH_URL="http://localhost:4000/api/v1/pm/task-statuses"
DB_ENV_FILE="$APP_DIR/.env"  # must contain POSTGRESQL_URL for psql below

mkdir -p "$APP_DIR" "$BACKUP_DIR"
TIMESTAMP="$(date -u +%Y%m%d%H%M%S)"

# 1. Backup current binary (if one exists).
if [ -f "$BIN_PATH" ]; then
  cp "$BIN_PATH" "$BACKUP_DIR/erp-cbqa-global.$TIMESTAMP"
  # Keep only the last 10 backups.
  ls -1t "$BACKUP_DIR"/erp-cbqa-global.* 2>/dev/null | tail -n +11 | xargs -r rm -f
fi

# 2. Apply migrations (additive-only, IF NOT EXISTS — safe to re-run).
if [ -d "$SRC_DIR/migrate" ] && [ -f "$DB_ENV_FILE" ]; then
  # shellcheck disable=SC1090
  source "$DB_ENV_FILE"
  if [ -n "${POSTGRESQL_URL:-}" ]; then
    for f in "$SRC_DIR"/migrate/*.sql; do
      [ -f "$f" ] || continue
      echo "Applying migration: $(basename "$f")"
      psql "$POSTGRESQL_URL" -f "$f" || echo "WARNING: migration $(basename "$f") returned non-zero (likely already applied)"
    done
  fi
fi

# 3. Swap binary and restart.
cp "$SRC_DIR/erp-cbqa-global" "$BIN_PATH"
chmod +x "$BIN_PATH"
systemctl restart "$SERVICE_NAME"

# 4. Health-check with retries.
echo "Waiting for $SERVICE_NAME to become healthy..."
for i in $(seq 1 15); do
  sleep 2
  if curl -sf -o /dev/null "$HEALTH_URL"; then
    echo "Health check passed."
    exit 0
  fi
done

echo "Health check FAILED after retries — rolling back."
LATEST_BACKUP="$(ls -1t "$BACKUP_DIR"/erp-cbqa-global.* 2>/dev/null | head -1 || true)"
if [ -n "$LATEST_BACKUP" ]; then
  cp "$LATEST_BACKUP" "$BIN_PATH"
  chmod +x "$BIN_PATH"
  systemctl restart "$SERVICE_NAME"
  sleep 3
  if curl -sf -o /dev/null "$HEALTH_URL"; then
    echo "Rollback succeeded."
  else
    echo "Rollback health check also failed — manual intervention required."
  fi
else
  echo "No backup available to roll back to — manual intervention required."
fi
exit 1
