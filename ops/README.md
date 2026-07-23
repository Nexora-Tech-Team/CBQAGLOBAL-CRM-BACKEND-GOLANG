# CI/CD — Go PM API

## Staging (push-based deploy)

Push to `staging` → `.github/workflows/deploy-staging.yml` builds the binary
on a GitHub-hosted runner, then pushes it directly to the staging box
(`212.85.25.165`, `api-pm-dev.nexoratech.co`) over SSH and runs
`deploy-golang-staging.sh` there (backup → migrate → swap binary → restart →
health-check → auto-rollback on failure). Push-based (unlike production,
below) because this repo only has collaborator-level GitHub access here too,
but the staging box accepts inbound SSH, so no polling/token dance is
needed — just SSH credentials as GitHub Secrets.

That box is **shared** with other projects (academy.nexoratech.co,
alpha.nexoratech.co, a Java backend + a frontend, MariaDB, PHP-FPM, and
their own self-hosted Actions runners) — the staging app lives in its own
`/root/app/golang-pm-staging` directory, its own systemd unit
(`cbqaglobal-golang-pm-staging`), and its own nginx vhost, so it shouldn't
interfere with those.

**Required GitHub configuration** (Settings → Environments → New environment
`staging`, then add these as environment secrets — never commit them):

| Secret | Value |
|---|---|
| `STAGING_SSH_HOST` | `212.85.25.165` |
| `STAGING_SSH_USER` | `root` |
| `STAGING_SSH_PASSWORD` | the VPS root password |

One-time server setup (already done for the current box, kept here for
reference / disaster recovery):

```bash
apt-get install -y postgresql-client
mkdir -p /root/app/golang-pm-staging
# .env seeded from this repo's .env.staging (POSTGRESQL_URL etc.)
cp ops/cbqaglobal-golang-pm-staging.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now cbqaglobal-golang-pm-staging
cp ops/nginx-api-pm-dev.nexoratech.co.conf /etc/nginx/sites-available/api-pm-dev.nexoratech.co
ln -s /etc/nginx/sites-available/api-pm-dev.nexoratech.co /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
certbot --nginx -d api-pm-dev.nexoratech.co
```

Verify: `systemctl status cbqaglobal-golang-pm-staging`,
`curl -s -o /dev/null -w '%{http_code}\n' https://api-pm-dev.nexoratech.co/api/v1/pm/task-statuses`
(expect 200). Logs: `tail -f /root/app/golang-pm-staging/service.log`.

Rollback: the deploy script auto-rolls back on a failed health check; to
roll back manually, copy the newest file from
`/root/app/golang-pm-staging/backups/` over `erp-cbqa-global` and
`systemctl restart cbqaglobal-golang-pm-staging`.

## Production — dedicated PM API box (push-based deploy)

Push to `main` → `.github/workflows/deploy-production.yml` builds the binary
on a GitHub-hosted runner, then pushes it directly to the dedicated PM API
production box (`72.60.74.36`, `api-pm-prod.cbqaglobal.co.id`) over SSH and
runs `deploy-golang-prod.sh` there (backup → migrate → swap binary →
restart → health-check → auto-rollback on failure). This box only hosts
this API plus the CRM frontend's own prod deploy — no shared-tenancy
concerns like the staging box above.

**Required GitHub configuration** (Settings → Environments → New environment
`production`, then add these as environment secrets — never commit them):

| Secret | Value |
|---|---|
| `PROD_SSH_HOST` | `72.60.74.36` |
| `PROD_SSH_USER` | `root` |
| `PROD_SSH_PASSWORD` | the VPS root password |

One-time server setup (already done for the current box, kept here for
reference / disaster recovery):

```bash
apt-get install -y postgresql-client
mkdir -p /root/app/golang-pm-prod
# .env seeded from this repo's .env.prod (POSTGRESQL_URL etc. — note the
# DB itself still lives on 72.60.74.35, this box just runs the API)
cp ops/cbqaglobal-golang-pm-prod.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now cbqaglobal-golang-pm-prod
cp ops/nginx-api-pm-prod.cbqaglobal.co.id.conf /etc/nginx/sites-available/api-pm-prod.cbqaglobal.co.id
ln -s /etc/nginx/sites-available/api-pm-prod.cbqaglobal.co.id /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
certbot --nginx -d api-pm-prod.cbqaglobal.co.id
```

Verify: `systemctl status cbqaglobal-golang-pm-prod`,
`curl -s -o /dev/null -w '%{http_code}\n' https://api-pm-prod.cbqaglobal.co.id/api/v1/pm/task-statuses`
(expect 200). Logs: `tail -f /root/app/golang-pm-prod/service.log`.

Rollback: the deploy script auto-rolls back on a failed health check; to
roll back manually, copy the newest file from
`/root/app/golang-pm-prod/backups/` over `erp-cbqa-global` and
`systemctl restart cbqaglobal-golang-pm-prod`.

## Legacy — shared box pull-based deploy (72.60.74.35)

Mirrors `CBQAGLOBAL-CRM-BACKEND`'s production deploy pattern: push to `main` builds a
binary on a GitHub-hosted runner and uploads it as an artifact; the VPS
(`72.60.74.35`, the same box that already serves `api-erp.cbqaglobal.co.id`) pulls
that artifact on a 2-minute systemd timer and deploys it locally. Pull-based because
this GitHub org access is collaborator-level, not org-admin, so a self-hosted Actions
runner cannot be registered here (same constraint documented in the Java backend's
`CLAUDE.md`).

## One-time setup on the VPS (run as root, or with sudo)

```bash
# 1. App directory
mkdir -p /root/app/golang-pm
cd /root/app/golang-pm

# 2. GitHub token for the poller (needs Contents:Read + Actions:Read on
#    Nexora-Tech-Team/CBQAGLOBAL-CRM-BACKEND-GOLANG). If the Java backend's
#    ci-pull-deploy already has a token with org-wide read access, reuse it:
cp /root/app/backend/.gh_token /root/app/golang-pm/.gh_token
chmod 600 /root/app/golang-pm/.gh_token
# ...otherwise generate a new fine-grained PAT scoped to this repo and place it here.

# 3. Runtime .env (same DB creds this repo already ships as .env.prod — copy
#    or recreate; PORT must be 4000 to match nginx/whatever proxies
#    api-erp.cbqaglobal.co.id today):
cat > /root/app/golang-pm/.env <<'EOF'
TZ=UTC
PORT=4000
GIN_MODE=release
POSTGRESQL_URL=postgresql://postgres:<PASSWORD>@72.60.74.35:5432/cbqaglobal?sslmode=disable
EOF
chmod 600 /root/app/golang-pm/.env

# 4. Copy the deploy scripts from this repo's ops/ directory
cp ops/ci-pull-deploy-golang.sh ops/deploy-golang.sh /root/app/golang-pm/
chmod +x /root/app/golang-pm/ci-pull-deploy-golang.sh /root/app/golang-pm/deploy-golang.sh

# 5. Install systemd units
cp ops/cbqaglobal-golang-pm.service /etc/systemd/system/
cp ops/ci-pull-deploy-golang.service ops/ci-pull-deploy-golang.timer /etc/systemd/system/
systemctl daemon-reload

# 6. IMPORTANT — if a Go process is already running manually (e.g. nohup) on
#    port 4000, stop it first so systemd owns the port:
#    pgrep -fl erp-cbqa-global   (find the manual PID, kill it, NOT with pkill)

# 7. Seed the binary once manually (first run has nothing to swap into yet),
#    then enable the service + timer:
#    (build the binary locally or trigger the GitHub Action once, download
#    the artifact, place it at /root/app/golang-pm/erp-cbqa-global)
systemctl enable --now cbqaglobal-golang-pm
systemctl enable --now ci-pull-deploy-golang.timer
```

## Verify

```bash
systemctl status cbqaglobal-golang-pm
systemctl status ci-pull-deploy-golang.timer
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:4000/api/v1/pm/task-statuses   # expect 200
curl -s -o /dev/null -w "%{http_code}\n" https://api-erp.cbqaglobal.co.id/api/v1/pm/task-statuses  # expect 200
tail -f /root/app/golang-pm/ci-pull-deploy-golang.log
```

## Manual controls

- Force a deploy check now: `systemctl start ci-pull-deploy-golang.service`
- Rollback: the deploy script auto-rolls back on a failed health check; to
  roll back manually, copy the newest file from `/root/app/golang-pm/backups/`
  over `erp-cbqa-global` and `systemctl restart cbqaglobal-golang-pm`.
- Watch logs: `tail -f /root/app/golang-pm/ci-pull-deploy-golang.log` and
  `tail -f /root/app/golang-pm/service.log`

## Files in this directory

| File | Purpose |
|---|---|
| `ci-pull-deploy-golang.sh` | Polls GitHub Actions for a new successful build, downloads the artifact, calls `deploy-golang.sh` |
| `deploy-golang.sh` | Backs up the current binary, applies new migrations, swaps the binary, restarts the service, health-checks, auto-rolls back on failure |
| `cbqaglobal-golang-pm.service` | systemd unit that actually runs the API |
| `ci-pull-deploy-golang.service` / `.timer` | systemd oneshot + timer that runs the poller every 2 minutes |
