#!/bin/bash
# Pull-based CI deploy poller for the Go PM API (erp-cbqa-global).
#
# Mirrors CBQAGLOBAL-CRM-BACKEND's ci-pull-deploy.sh: this VPS only has
# collaborator (not org-admin) GitHub access, so a self-hosted Actions
# runner can't be registered. Instead this script runs on a systemd timer
# and PULLS the latest successful GitHub Actions build over HTTPS
# (outbound-only — never needs inbound SSH from GitHub).
#
# State lives in $STATE_DIR:
#   .gh_token           GitHub PAT (repo + actions:read), chmod 600.
#                        Reuse the Java backend's token if it already has
#                        access to this repo (same org) — copy it here or
#                        symlink; otherwise generate a fine-grained PAT
#                        scoped to Nexora-Tech-Team/CBQAGLOBAL-CRM-BACKEND-GOLANG
#                        with Contents:Read + Actions:Read.
#   .last_deployed_sha  SHA of the last successfully deployed build.
#   .failed_sha         SHA of the last build that failed to deploy —
#                        skipped until a newer commit lands, to avoid
#                        flapping on a broken commit every 2 minutes.
#   ci-pull-deploy-golang.log   Rolling log of every poll.

set -euo pipefail

REPO="Nexora-Tech-Team/CBQAGLOBAL-CRM-BACKEND-GOLANG"
BRANCH="main"
WORKFLOW_FILE="build.yml"
ARTIFACT_NAME="golang-pm-api"

STATE_DIR="/root/app/golang-pm"
TOKEN_FILE="$STATE_DIR/.gh_token"
LAST_SHA_FILE="$STATE_DIR/.last_deployed_sha"
FAILED_SHA_FILE="$STATE_DIR/.failed_sha"
LOG_FILE="$STATE_DIR/ci-pull-deploy-golang.log"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" >> "$LOG_FILE"; }

mkdir -p "$STATE_DIR"
touch "$LOG_FILE"

if [ ! -f "$TOKEN_FILE" ]; then
  log "ERROR: missing $TOKEN_FILE (GitHub PAT). Aborting."
  exit 1
fi
GH_TOKEN="$(cat "$TOKEN_FILE")"

AUTH_HEADER="Authorization: Bearer $GH_TOKEN"
API="https://api.github.com/repos/$REPO"

# 1. Find the latest successful run of build.yml on $BRANCH.
RUN_JSON="$(curl -sf -H "$AUTH_HEADER" -H "Accept: application/vnd.github+json" \
  "$API/actions/workflows/$WORKFLOW_FILE/runs?branch=$BRANCH&status=success&per_page=1")"

RUN_ID="$(echo "$RUN_JSON" | python3 -c 'import sys,json; d=json.load(sys.stdin); r=d.get("workflow_runs") or []; print(r[0]["id"] if r else "")')"
HEAD_SHA="$(echo "$RUN_JSON" | python3 -c 'import sys,json; d=json.load(sys.stdin); r=d.get("workflow_runs") or []; print(r[0]["head_sha"] if r else "")')"

if [ -z "$RUN_ID" ]; then
  log "No successful run found for $WORKFLOW_FILE on $BRANCH. Skipping."
  exit 0
fi

LAST_SHA="$(cat "$LAST_SHA_FILE" 2>/dev/null || echo "")"
FAILED_SHA="$(cat "$FAILED_SHA_FILE" 2>/dev/null || echo "")"

if [ "$HEAD_SHA" = "$LAST_SHA" ]; then
  exit 0  # already deployed, nothing to do — quiet, no log spam
fi
if [ "$HEAD_SHA" = "$FAILED_SHA" ]; then
  exit 0  # known-bad commit, wait for a newer one
fi

log "New build detected: run=$RUN_ID sha=$HEAD_SHA (current=$LAST_SHA)"

# 2. Download the artifact.
ARTIFACTS_JSON="$(curl -sf -H "$AUTH_HEADER" -H "Accept: application/vnd.github+json" \
  "$API/actions/runs/$RUN_ID/artifacts")"
ARTIFACT_URL="$(echo "$ARTIFACTS_JSON" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for a in d.get('artifacts', []):
    if a['name'] == '$ARTIFACT_NAME':
        print(a['archive_download_url'])
        break
")"

if [ -z "$ARTIFACT_URL" ]; then
  log "ERROR: artifact '$ARTIFACT_NAME' not found on run $RUN_ID."
  echo "$HEAD_SHA" > "$FAILED_SHA_FILE"
  exit 1
fi

curl -sfL -H "$AUTH_HEADER" -o "$TMP_DIR/artifact.zip" "$ARTIFACT_URL"
unzip -q "$TMP_DIR/artifact.zip" -d "$TMP_DIR/extracted"

if [ ! -f "$TMP_DIR/extracted/erp-cbqa-global" ]; then
  log "ERROR: binary missing after extract."
  echo "$HEAD_SHA" > "$FAILED_SHA_FILE"
  exit 1
fi
chmod +x "$TMP_DIR/extracted/erp-cbqa-global"

# 3. Deploy (backup + swap + restart + health-check + auto-rollback).
if bash "$STATE_DIR/deploy-golang.sh" "$TMP_DIR/extracted" >> "$LOG_FILE" 2>&1; then
  echo "$HEAD_SHA" > "$LAST_SHA_FILE"
  rm -f "$FAILED_SHA_FILE"
  log "Deploy succeeded: sha=$HEAD_SHA"
else
  echo "$HEAD_SHA" > "$FAILED_SHA_FILE"
  log "Deploy FAILED: sha=$HEAD_SHA — see log above. Will not retry until a newer commit lands."
  exit 1
fi
