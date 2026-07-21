# CBQAGLOBAL CRM — Go PM Backend — Claude Context

## Project Overview

Go (Gin + GORM) backend serving the **Project Management (PM)** module of the
CBQAGLOBAL CRM. Sibling repos: main CRM/Audit/HRIS backend is Spring Boot
(`CBQAGLOBAL-CRM-BACKEND`), frontend is React (`CBQAGLOBAL-CRM-FRONTEND`).
This service only owns PM — everything else lives in the Java backend.

---

## Stack

- Go, module `erp-cbqa-global`, Gin, GORM — but almost everything goes
  through raw SQL via `.Raw(...).Scan(&rows)` into `model.Row =
  map[string]interface{}`, not GORM struct models. Check
  `domain/pm/repository/repository.go` for the actual query shapes before
  assuming a field exists.
- PostgreSQL — **same physical database** as the Java CRM backend
  (`cbqaglobal`). PM-owned tables are prefixed `pm_`, but this service also
  reads/writes CRM tables directly (`projects`, `leads`, `companies`,
  `users`) for the CRM-linked PM workspace.
- **No auth** on `/api/v1/pm/*` yet (see the comment in the frontend's
  `src/apiPm.js`) — anyone who can reach the host can call these endpoints.
  Don't add anything sensitive without fixing this first.

---

## Two "PM projects" concepts — don't confuse them

- **Legacy `pm_projects` table** (the "Library" module) — independent
  projects keyed by their own `legacy_id`, not tied to CRM.
- **CRM-linked PM ("PM baru")** — real CRM `projects` rows (Advisory/Audit
  Program service) shown at `/pm/projects` in the frontend. A `pm_projects`
  row is created **lazily** on first use (`ensurePmProject` pattern — see
  `UpsertPmProjectPic`) via `pm_projects.crm_project_id`, used purely as a
  "shadow row" to hold PM-only fields (currently just `pic_user_id`) that
  don't belong on the CRM `projects` table itself.
- `pm_projects.legacy_id` (**not** `id`) is the FK target for
  `pm_tasks.project_id` — legacy quirk, don't "fix" it without checking both
  call sites.

## Endpoints for CRM-linked PM

| Method | Path | Repo/Service function |
|---|---|---|
| GET | `/api/v1/pm/crm-projects` | `CrmProjects` — list |
| GET | `/api/v1/pm/crm-projects/{id}` | `CrmProjectByID` / `CrmProjectDetail` — detail (+ tasks/team/activity) |
| PUT | `/api/v1/pm/crm-projects/{id}/overview` | `UpdateCrmProjectOverview` — owner, project dates, PIC |
| GET | `/api/v1/pm/crm-projects/{id}/tasks` | `ProjectTasksByCrmProject` |
| GET/POST | `/api/v1/pm/crm-projects/{id}/members` | `TeamByCrmProject` / add member |
| GET | `/api/v1/pm/gantt/members` | Internal users, scoped to department "IT Audit" |

---

## Stage — computed, never manual (added 2026-07-21)

`stage` on both `CrmProjects` (list) and `CrmProjectByID` (detail) is derived
purely from each project's active (`deleted = FALSE`) `pm_project_tasks` —
there is no manual "set stage" input anywhere in the system.

The SQL lives once, as two `const` fragments in `repository.go`, embedded in
both queries so list and detail can never disagree:

- `taskStageJoinSQL` — aggregates task counts (`total/blocked/done/review/progress`)
  per `crm_project_id` in one grouped subquery (no N+1 — a single extra join,
  not a per-row lookup).
- `taskStageCaseSQL` — priority CASE: **Blocked > Completed > Review >
  Fieldwork > Planning**. `Completed` only fires when ALL active tasks are
  done (and at least one exists); zero active tasks is always `Planning`
  regardless of any other signal.

Real task status values in `pm_project_tasks.status` (cross-check against
`pm_task_statuses` before trusting this list): `to_do`, `in_progress`,
`in_review`, `blocked`, `done`. `blocked` is seeded as a selectable Kanban
column, so moving any active task there makes the project Stage become
`Blocked` until that task leaves the blocked status.

`domain/pm/service/service.go`'s `CrmProjectDetail` used to **override** the
SQL's `stage` with a stale 2-state approximation (`Planning`/`Fieldwork`/`Closed`
derived from task-completion %) — this override has been removed. `progress`
(task-completion percentage) is a **separate** metric from `stage` and is
still computed there; don't conflate the two when touching this function.

## PIC (Person In Charge) — added 2026-07-21

`pm_projects.pic_user_id` → joined to `users` in both `CrmProjects` and
`CrmProjectByID` as `pic` (name) / `pic_user_id`. Set via
`PUT /crm-projects/{id}/overview` → `UpsertPmProjectPic`, which
auto-creates the shadow `pm_projects` row if none exists yet (same
lazy-creation pattern as `ensurePmProject`).

---

## Local dev

```bash
go run main.go   # PORT=4000; .env already points POSTGRESQL_URL at the
                  # STAGING db (45.13.132.234) — there is no separate local DB
```

**Gotcha — `go run` leaves a zombie child on restart.** `go run` spawns a
child process (the actual compiled binary, e.g.
`/var/folders/.../go-build.../exe/main` or `~/Library/Caches/go-build/...`)
that is a **separate PID** from the `go run` wrapper. Killing the wrapper
does **not** kill the child — it keeps listening on port 4000 and silently
serves whatever code was compiled at the moment it started, even after you
edit and save new source. Confirmed twice in this repo's history: a source
edit sat on disk for ~1 hour while the stale child kept answering requests
with pre-edit behavior, and a later restart attempt killed only the wrapper,
leaving the *previous* stale child still bound to the port. Always fully
verify before trusting a "restarted" local server:

```bash
pkill -9 -f "go run main.go"
lsof -i :4000 -sTCP:LISTEN -t | xargs -r kill -9   # kill the actual child too
lsof -i :4000 -sTCP:LISTEN                          # must print nothing
go run main.go
```

---

## Deploy

Full CI/CD detail — required GitHub secrets, systemd units, one-time VPS
setup, rollback commands — lives in `ops/README.md`. Summary:

- **Staging**: push to `staging` → push-based deploy over SSH to
  `212.85.25.165` (`api-pm-dev.nexoratech.co`). See
  `.github/workflows/deploy-staging.yml` + `ops/deploy-golang-staging.sh`.
- **Production**: push to `main` → builds an artifact; a separate box
  (`72.60.74.35`, `api-erp.cbqaglobal.co.id`) polls and pulls it every 2 min
  via `ops/ci-pull-deploy-golang.sh` (collaborator-level GitHub access here,
  can't register a self-hosted runner for push-based deploy).
- **Different databases** — staging → `45.13.132.234`; production →
  `72.60.74.35`. Don't assume data is in sync between them.

### Fixed: "Text file busy" on binary swap (2026-07-21)

Both `deploy-golang-staging.sh` and `deploy-golang.sh` used to `cp` the new
binary directly over the running one — this fails with `Text file busy`
because `systemctl restart` only happens *after* the copy, while the old
process still has the binary mapped into memory. Fixed by copying to
`$BIN_PATH.new` then `mv` (atomic rename) in both the deploy and rollback
paths — the kernel allows swapping a directory entry to a new inode even
while the old inode is still mapped by a running process.

### Fixed: migration 001 not idempotent (2026-07-21)

`001_create_invoices_tables.sql` was the only migration without
`IF NOT EXISTS` (every migration from 002 onward has it) — threw
`relation already exists` on every redeploy. Non-fatal (the deploy script
treats a migration failure as a warning, not an abort) but noisy enough to
mask a real migration error; fixed to match the others.

---

## Frontend integration gotcha — staging FE must set REACT_APP_PM_API_ENDPOINT (2026-07-21)

`CBQAGLOBAL-CRM-FRONTEND/src/apiPm.js` falls back to a **hardcoded
production URL** (`https://api-erp.cbqaglobal.co.id`) whenever
`REACT_APP_PM_API_ENDPOINT` isn't set at build time. The staging frontend
build (`deploy.yml`) never injected this var, so `alpha.nexoratech.co` was
silently talking to the **production** Go backend (different DB —
`72.60.74.35`, not staging's `45.13.132.234`) for the entire PM module.
Result: PIC and Stage, both deployed correctly to staging, never appeared
on the staging frontend no matter how many times staging was redeployed —
because the frontend was never actually calling staging's Go backend.
Fixed by adding `REACT_APP_PM_API_ENDPOINT=https://api-pm-dev.nexoratech.co`
directly to the frontend repo's `.env.staging` (safe to commit — it's a
plain URL, not a secret, unlike `REACT_APP_API_ENDPOINT` which must only
ever come from the `STAGING_API_URL` GitHub secret — see the frontend
repo's `CLAUDE.md` for why mixing the two breaks `dotenv` precedence).

**Rule of thumb:** if a staging PM feature "isn't showing up" after a
confirmed-successful deploy, check which Go backend host the frontend is
actually calling (Network tab / build's baked-in `apiPm` baseURL) before
debugging the backend further — both hosts respond 200 on the same route
shapes, so this kind of cross-wiring is easy to miss.
