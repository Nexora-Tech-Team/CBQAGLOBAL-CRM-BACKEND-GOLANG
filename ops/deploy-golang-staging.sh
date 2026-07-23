#!/bin/bash
# Deploys a new erp-cbqa-global binary to the STAGING box (212.85.25.165,
# api-pm-dev.nexoratech.co): backup current binary, swap the binary,
# restart the systemd service, health-check it, auto-rollback on failure.
# Migrations are applied manually direct to the DB, not by this script.
#
# Usage: deploy-golang-staging.sh <path-to-extracted-artifact-dir>
#   (that dir must contain ./erp-cbqa-global)
#
# Runs on the server itself, invoked over SSH by
# .github/workflows/deploy-staging.yml (push-based — this box only has
# collaborator-level GitHub access for this repo, and is shared with other
# projects' self-hosted runners, so we push over SSH instead).

set -euo pipefail

SRC_DIR="${1:?Usage: deploy-golang-staging.sh <extracted-artifact-dir>}"
APP_DIR="/root/app/golang-pm-staging"
BIN_PATH="$APP_DIR/erp-cbqa-global"
BACKUP_DIR="$APP_DIR/backups"
SERVICE_NAME="cbqaglobal-golang-pm-staging"
HEALTH_URL="http://localhost:4000/api/v1/pm/task-statuses"

mkdir -p "$APP_DIR" "$BACKUP_DIR"
TIMESTAMP="$(date -u +%Y%m%d%H%M%S)"

# 1. Backup current binary (if one exists).
if [ -f "$BIN_PATH" ]; then
  cp "$BIN_PATH" "$BACKUP_DIR/erp-cbqa-global.$TIMESTAMP"
  # Keep only the last 10 backups.
  ls -1t "$BACKUP_DIR"/erp-cbqa-global.* 2>/dev/null | tail -n +11 | xargs -r rm -f
fi

# 2. Migrations are applied manually direct to the DB, not from here.

# 3. Swap binary and restart. Copy to a temp file then rename into place —
# an in-place overwrite (cp truncating the target) fails with "Text file
# busy" while the old binary is still running.
TMP_BIN="$BIN_PATH.new"
cp "$SRC_DIR/erp-cbqa-global" "$TMP_BIN"
chmod +x "$TMP_BIN"
mv "$TMP_BIN" "$BIN_PATH"
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
  cp "$LATEST_BACKUP" "$TMP_BIN"
  chmod +x "$TMP_BIN"
  mv "$TMP_BIN" "$BIN_PATH"
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
