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

## PM Timesheet source of truth (2026-07-23)
The PM Timesheet endpoint GET /api/v1/pm/timesheets must read from pm_task_time_logs,
not the legacy pm_timesheets table. The task drawer Work Logs, dashboard active sessions, and
Timesheet page now share the same source of truth: realtime Clock In/Clock Out rows and manual
work logs in pm_task_time_logs. Repository Timesheets joins pm_task_time_logs to pm_project_tasks,
projects/leads/companies, and users to return task/subtask, parent task, project title, member,
start/end, duration_seconds, source, and note. Cast rounded hour values to float8 in SQL so JSON
contains numbers (e.g. 0.38) instead of PostgreSQL numeric bytes/base64 from model.Row scanning.
If /pm/timesheet is empty while a task drawer shows logs, restart/deploy this Go service first;
that symptom usually means an old process is still serving the old pm_timesheets query.

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

## Work Timer (clock in/out) + Manual Work Log (2026-07-22)

`pm_task_time_logs` (migration 013, `source`/`created_by`/`updated_by` added
in 014) backs both real-time clock sessions and manually-entered ones — see
`timeLogSelectSQL` in `repository.go` for the single shared read shape.

- **Real-time**: `POST /tasks/:id/clock-in` / `clock-out`. A user may have
  exactly one OPEN session (`ended_at IS NULL`) at a time, across ALL tasks —
  enforced by a UNIQUE partial index (`uq_pm_task_time_logs_one_active_per_user`),
  not just the service-layer check-then-insert (which only exists to turn a
  race into a clean `ErrAlreadyClockedIn` 409 instead of a raw constraint
  error). `GET /time-logs/active?userId=` answers "is this user clocked in
  anywhere" as a flat shape (`{active:false}` or `{active, taskId, taskTitle,
  taskType: "task"|"subtask", parentTaskId, parentTaskTitle, projectId,
  projectTitle, startedAt, elapsedSeconds}`) — enriched with parent/project
  titles in one call so the frontend never needs a second lookup just to
  render "Clocked in on another task: X".
- **Manual**: `POST /tasks/:id/time-logs/manual` (body: `userId, startedAt,
  endedAt, note` — note is mandatory, `startedAt < endedAt` enforced), `PATCH
  /time-logs/:id`, `DELETE /time-logs/:id` — the latter two only operate on
  `source = 'manual'` rows (a real-time session's timestamps are a factual
  record, not something to hand-edit). `ManualLogOverlapExists` rejects a
  manual entry that overlaps ANY of the user's existing sessions — closed
  ones by their stored `ended_at`, an open (active clock-in) one treated as
  extending to `'infinity'` — via a single `tsrange(...) && tsrange(...)`
  query, so a manual entry can never silently double-book time already being
  tracked live.
- **Known gap, deliberate**: the spec's "regular users can only log their own
  time; Admin/PM can log for others" isn't enforced server-side — this PM API
  group has no auth at all yet (see the top of this file), so there's no
  reliable role signal to check against here. `created_by`/`updated_by` do
  still record who actually performed the write (vs `user_id`, who the log
  is *for*), and the activity-log description names both when they differ.

## Parent task status — now derived, same pattern as progress (2026-07-22)

`computeTaskProgress`'s `taskProgress` struct gained a `Status` field
alongside `Progress`. A parent's status is `deriveParentStatus` over its
children's own *effective* status (recursive — a nested parent contributes
its derived status, not its raw stored one): priority Blocked > In Review >
In Progress > Done > To Do, where "Done" only fires once *every* child has
status `done` or effective progress 100 — never partially. `applyTaskProgress`
overrides a parent row's `status`/`status_key`/`status_title`/`status_color`
in place (the title/color come from a small `statusMeta` map mirroring the
frontend's `KANBAN_COLS`, since the row never went through the real
`pm_task_statuses` join for this derived value) — same "always compute at
read time, never trust the stored column" architecture as progress.

`UpdateProjectTask` and `MoveProjectTaskByKey` (Kanban drag) both now guard
the status write the same way they already guarded progress: a manual
status change aimed at a parent task is silently dropped (update) or turns
the whole drag into a no-op on both stored columns (move), never an error —
the frontend should prevent dragging a parent card in the first place, but
the backend is the actual source of truth.

**Known limitation, deliberate**: Stage (`taskStageCaseSQL`) still counts
every active task by its *raw stored* `status` column, including parent
rows — it does not read through this new derivation. A parent whose stored
status has gone stale relative to its children could very slightly skew
Stage's task-status counts. Not fixed here: doing this correctly in SQL
would need the same bottom-up recursion `computeTaskProgress` does in Go,
which is a bigger change than this task's scope (renaming the Stage label,
not re-deriving its inputs). Flagging for whoever touches Stage next.

## PM Dashboard summary endpoint — `GET /v1/pm/dashboard/summary` (2026-07-22)

New high-level portfolio monitoring endpoint powering `/pm/dashboard` on the
frontend (project health, portfolio progress, team workload, active work
sessions, upcoming deadlines, recent activity). Route registered in
`config/collection/routes.go`; handler `pmCtrl.DashboardSummary` →
`service.DashboardSummary()` → 6 new repository methods, all documented
inline in `repository.go`/`service.go`:

- `PmPortfolioProjects()` — one row per CRM-linked PM project (same
  Advisory/Audit Program scope as `CrmProjects`), reusing `taskStageCaseSQL`/
  `taskStageJoinSQL` verbatim for Stage (never re-derived differently), plus
  a new `planWindowJoinSQL` const (`MIN(start_date)`/`MAX(deadline)` per
  project) for the planned-progress side of Portfolio Progress.
- `PmPortfolioTasks()` — every non-deleted PM task, unscoped (same "small
  table, fetch once" rationale as the existing `ActiveTaskProgressInputs`),
  widened with `title`/`deadline`/`assigned_to`/`assignee_name` for
  workload/overdue/deadline derivation.
- `PmMembersForWorkload()` — `GanttMembers`' exact WHERE clause (IT Audit
  dept, active internal users) plus `job_title`, as its own method so
  `GanttMembers` itself (the assignee picker) can't drift.
- `PmActiveWorkSessions()` — every currently open (`ended_at IS NULL`) time
  log across ALL users (not just one, unlike `ActiveTimeLogForUser`) — the
  org-wide "who's clocked in right now" view, also used to fill each Team
  Workload row's `currentClockIn`.
- `PmWeeklySecondsByUser()` — per-user seconds tracked this calendar week
  (`date_trunc('week', NOW())`, matching `TimesheetSummary`'s own
  week-bucketing convention), including any still-open session's live
  elapsed time.
- `PmRecentActivity(limit)` — `TaskActivityLogs` widened from "one task" to
  "every task, most recent first". There is currently no project-level
  "project updated" / "member added" log entry anywhere in this codebase
  (`UpdateCrmProject`/`InsertCrmProjectMember` don't call
  `logTaskActivity`), so those two categories from the frontend spec simply
  won't appear in the feed yet — never fabricated to fill the gap.

**New derivation rules, all in `service.go` (no prior metric existed for
these — first-principles, documented inline):**

- **Planned progress** (`planProgressPercent`): % of a project's
  `plan_start`→`plan_end` window elapsed as of now, clamped 0-100. Falls
  back to 0 (not excluded) when either bound is missing, so portfolio
  averages always divide by every project.
- **Actual progress**: reuses `computeTaskProgress`/`projectProgressFromTasks`
  verbatim — same source of truth as `CrmProjects`/`CrmProjectDetail`.
- **Health**: Blocked (any blocked task or Stage=Blocked) > At Risk (any
  overdue task, or plan window's end already past and Stage≠Completed, or
  actual < planned) > Healthy.
- **Attention Needed issue** (single most severe per project): Blocked >
  Overdue > Behind plan > No recent update (`updated_at` > 14 days old).
  Capped at 10 rows, most critical first.
- **Overdue** (`isOverdueTask`) and the Upcoming Deadlines Overdue/Due
  Today/Due This Week split both compare **calendar days**, not exact
  timestamps (`isPastCalendarDay`/`isSameCalendarDay`) — task deadlines are
  stored as midnight-of-day values, so an exact `Before(now)` check would
  make "Due Today" unreachable (a deadline stored at today's 00:00 is
  "before now" the instant any time at all has passed today). Bug caught
  and fixed during initial smoke testing — see the same rule applied to
  `planEndPassed` in the Health computation for consistency.
- **Load status** (`loadStatus`): Berat (active>5 or hours>35 or overdue>0)
  > Sedang (active 3-5 or hours 20-35) > Ringan — checked heaviest-first so
  any single qualifying condition escalates.

**Graceful-empty-data guarantees** (per the task's explicit requirement):
every repository call in `DashboardSummary()` past the first
(`PmPortfolioProjects`) degrades to an empty slice on error rather than
failing the whole response; every numeric field defaults to 0 via Go's
zero-value semantics (no manual "if empty" branching needed) — confirmed
against both a project with real tasks/time-logs and (by code review, no
panics possible: every map/slice access is nil-safe, every division is
length-guarded) the true zero-data case.

Smoke-tested against the shared staging DB (2026-07-22): `curl
localhost:4000/api/v1/pm/dashboard/summary` — 8 real PM projects, correct
Stage/Health tallies, `attentionNeeded` correctly ranked Blocked-first, no
active work sessions (empty array, no error), `recentActivity` showing real
task-activity-log rows (created/status changes/manual work logs).

## Team Workload period filter — `GET /v1/pm/dashboard/team-workload` (2026-07-22)

Follow-up to the dashboard summary endpoint above: `DashboardSummary`'s
`teamWorkload` is always "this calendar week" (via `PmWeeklySecondsByUser`)
and can't be repointed at a different period without refetching the whole
dashboard. This new, separate endpoint gives the Team Workload section its
own period filter without touching any other section — a dedicated
endpoint was chosen over an `?workloadPeriod=` query param on
`/dashboard/summary` specifically so switching period only ever issues one
small request, not a full-dashboard refetch.

`GET /dashboard/team-workload?period=this_week|this_month|custom_month[&month=YYYY-MM]`
— `period` defaults to `this_week`; `month` is required (and validated,
`400` via `service.ErrInvalidWorkloadPeriod` if missing/malformed) only for
`custom_month`.

```json
{
  "period": { "type": "this_month", "label": "July 2026", "startDate": "...", "endDate": "...", "workdaysElapsed": 16 },
  "teamWorkload": [ { "userId": 41, "memberName": "...", "jobTitle": "...", "activeTasks": 2, "inProgressTasks": 0, "completedTasks": 1, "overdueTasks": 0, "hoursLogged": 0, "avgHoursPerDay": 0, "currentClockIn": null, "loadStatus": "Ringan" } ]
}
```

### Which fields are period-scoped vs always-current

Per the explicit requirement ("agar dashboard tetap actionable"):
- **Always CURRENT, never historical**: `activeTasks`, `inProgressTasks`,
  `overdueTasks` (all three: `PmPortfolioTasks()`'s raw stored `status`/
  `deadline`, same as `DashboardSummary`), and `currentClockIn`
  (`PmActiveWorkSessions()` — a live open session doesn't have a "period",
  it either exists right now or it doesn't). A June workload report still
  shows what's on someone's plate *today*, by design.
- **Scoped to the selected period**: `completedTasks` and `hoursLogged`
  (and `avgHoursPerDay`, derived from `hoursLogged`).

### Period resolution (`resolvePeriodRange`, `service.go`)

| `period` | `start` | `end` | ongoing? |
|---|---|---|---|
| `this_week` | Monday 00:00 of the current ISO week | `now` | always |
| `this_month` | 1st of the current month 00:00 | `now` | always |
| `custom_month` (= current month) | 1st of that month 00:00 | `now` | yes |
| `custom_month` (past or future) | 1st of that month 00:00 | 1st of the *following* month 00:00 (full calendar month) | no |

"Ongoing" controls `workdaysElapsed` (below) and is returned from
`resolvePeriodRange` as an explicit `isOngoing bool` — deliberately not
re-derived by comparing `end` to `now` at the call site (fragile,
`time.Time` equality isn't a safe signal), the resolver just tells the
caller directly.

### Hours Logged — "the safest option", per the task's own framing

`PmWorkloadSecondsByUserForRange(start, end)`: sums `duration_seconds` for
`pm_task_time_logs` rows with `ended_at IS NOT NULL` (i.e. **closed**
sessions only — both finished realtime clock sessions and manual work logs,
since a manual entry is always inserted already-closed) whose `started_at`
falls in `[start, end)`. **A still-open (active) session's live elapsed
time is deliberately NOT added** — unlike `PmWeeklySecondsByUser` (the
always-"this week" summary card, which DOES fold in the running session for
a "what's happening right now" feel), a period report needs to be a stable,
reproducible number. Including live elapsed time would make `hoursLogged`
silently tick upward every second while someone has the dashboard open,
which reads as a bug for a report, not a feature. Once the user clocks out,
that session's real duration lands in the period it was closed... no —
lands in the period it *started* in (`started_at`-bucketed, matching every
other time-bucketing convention in this codebase, e.g. `TimesheetSummary`).

### Average Hours / Day — workday counting, never divides by zero

`workdaysElapsed` = count of Mon-Fri calendar days (`countWeekdays`) in the
period, capped at "up to and including today" when `isOngoing`, or the full
month when a past `custom_month` (`end` is already the exclusive month
boundary in that case, so no `now` cap is applied). `avgHoursPerDay =
hoursLogged / workdaysElapsed`, but **only when `workdaysElapsed > 0`** —
otherwise `0`. This can only be zero in one real edge case: viewing
`this_week`/`this_month` on a Saturday/Sunday that falls before the first
weekday of that period has occurred yet (impossible for a past
`custom_month`, since a full real month always contains at least one
weekday). `workdaysElapsed` itself is still reported honestly as `0` in the
response — only the division is guarded, the underlying data isn't hidden.

### Load Status — weekly vs monthly thresholds

`loadStatusForPeriod(activeTasks, overdueTasks, hoursLogged, periodKind)` —
a period-aware sibling of the always-weekly `loadStatus` used by
`DashboardSummary` (kept as two separate functions on purpose: the summary
card's path stays simple and can't regress if this one changes).
`periodKind` is `"weekly"` for `this_week`, `"monthly"` for
`this_month`/`custom_month` (any month-based period uses monthly
thresholds, including the current month via `custom_month`):

| | Ringan | Sedang | Berat |
|---|---|---|---|
| activeTasks | ≤2 | 3-5 | >5 |
| hoursLogged (weekly) | <20 | 20-35 | >35 |
| hoursLogged (monthly) | <80 | 80-140 | >140 |
| overdueTasks | 0 | — | >0 (forces Berat regardless of the above) |

Checked heaviest-first (Berat, then Sedang, else Ringan) so any single
qualifying condition escalates — e.g. zero active tasks but 150 logged
hours this month still reports Berat.

### Completed Tasks — best-effort via `status_changed_at`

`PmCompletedTasksByUserForRange`: counts each assignee's tasks with
`LOWER(status) IN ('done','completed')` whose `status_changed_at` falls in
`[start, end)` — the same column `MoveProjectTaskByKey`/`UpdateProjectTask`
already stamp on every transition, no new column added. Best-effort since
that column only holds the *latest* transition: a task Done→reopened→Done
again within one window still counts once (correct); a task completed in a
past period and never touched again correctly stops counting once the
period moves past it.

Frontend: `PmDashboard/index.jsx` — segmented control (This Week/This
Month/Custom Month) in the Team Workload card header, native
`<input type="month">` for Custom Month. Switching period calls only
`GET /dashboard/team-workload` (confirmed via network trace during manual
testing — no other section refetches). See the frontend repo's `CLAUDE.md`
for the UI-side detail.

## Stage label: 'Fieldwork' renamed to 'In Progress' (2026-07-22)

`taskStageCaseSQL`'s progress-tasks branch now emits `'In Progress'` instead
of `'Fieldwork'` — same priority order (Blocked > Completed > Review > In
Progress > Planning), same trigger condition, label only. See the frontend
repo's `CLAUDE.md` for the matching `PROJECT_STAGES`/`normalizeStageLabel`
change (the latter rewrites any lingering `'Fieldwork'` value read-time, so
a stale cached response never surfaces the retired label).

## `CrmProjects` (the `/pm/projects` list) now computes real Health (2026-07-23)

`GET /api/v1/pm/crm-projects` previously never set a `health` field at all —
the frontend's `normalizeProject()` fell back through
`project.health || project.statusHealth || project.projectHealth ||
project.status`, and since none of the first three ever existed, every row
silently showed the raw CRM `projects.status` integer (always `2` for every
PM-linked project, since that's the "Won"/converted status code) with a
default green badge. Not a frontend bug — the list endpoint just never
computed Health, unlike `DashboardSummary` which always has.

**Fix**: `CrmProjects` (repository + service) now computes Health the exact
same way `DashboardSummary` does, via a new shared function,
**`deriveProjectHealth(stage, blockedTasks, overdueTaskCount, planEndPassed,
actualProgress, plannedProgress) string`** (`service.go`) — Blocked > At
Risk > Healthy:
- **Blocked**: any active task is blocked, or Stage is already `"Blocked"`.
- **At Risk**: any task is overdue, OR the plan window's end date has
  already passed while Stage isn't `"Completed"`, OR actual progress (real
  task completion %) is behind planned progress (where the plan window's
  start/end date says the project should be by now — see
  `planProgressPercent`).
- **Healthy**: none of the above.

`DashboardSummary`'s own inline health `switch` (previously duplicated
inline) was refactored to call `deriveProjectHealth` too, so the two
surfaces structurally cannot drift apart again.

**Wiring**: `CrmProjects`' repository SQL gained `blocked_tasks` (from the
already-joined `taskStageJoinSQL`'s `task_stage` aggregate — was joined but
never selected) and `plan_start`/`plan_end` (via `planWindowJoinSQL`, same
COALESCE-to-`project_date`/`valid_until` fallback `PmPortfolioProjects`
already uses for brand-new Planning projects with no tasks yet). The
service layer switched its per-project task source from
`ActiveTaskProgressInputs()` to **`PmPortfolioTasks()`** — Health's overdue
check needs each task's `deadline`, which `ActiveTaskProgressInputs` never
selected (it only has `id`/`crm_project_id`/`parent_task_id`/`status`/
`progress_pct`, enough for progress but not for overdue detection).
`ActiveTaskProgressInputs` itself is left in place (still compiles, no
other caller currently, but not worth deleting for a one-line savings).

**Scope note**: only the **list** endpoint (`CrmProjects`) was fixed, since
that's what `/pm/projects` calls and what was reported. `CrmProjectByID`
(`/pm/projects/{id}` detail page) has the exact same gap — its `health`
field is likewise never set by that query — but was deliberately left
alone; fix it the same way (add `blocked_tasks`/`plan_start`/`plan_end` to
its SQL, compute via `deriveProjectHealth` in `CrmProjectDetail`) if the
detail page's Health badge needs the same treatment.

Verified against the shared staging DB (2026-07-23): before the fix, every
row returned `status: 2` and no `health` key at all; after, real projects
resolved to `Healthy` (e.g. project 994, all tasks on track) and `At Risk`
(e.g. project 827, several Planning-stage projects with no tasks yet whose
plan window has already started) — no `Blocked` example in the current
dataset, but the same-signal logic as `DashboardSummary`'s already-verified
Health tallies.

## `leafProgressForStatus` — final confirmed rule table, after 2 correction rounds (2026-07-23)

A Done → In Review transition was reported showing `progress_pct: 100`
still, on task `Collect Document` (project 994) via the Task Detail
drawer's Edit form. This function went through three revisions the same
day before landing on the rule actually wanted — recorded here so the next
change doesn't repeat the same back-and-forth:

1. **First attempt**: forced 100% to be exclusive to Done/Completed —
   `in_review` snapping to 90 whenever it arrived at/above 100, plus a
   blanket cap (100 → 99) on `blocked`/`in_progress`/`default` too. Wrong:
   broke Done → In Progress (showed 99% instead of the expected 100%) and
   would have broken Done → Blocked the same way.
2. **First revert**: back to the function's original logic — `in_progress`
   and `in_review` both pure floors that never lower an already-higher
   value, `blocked` never touched. Matched the user's own stated rule
   table at the time... except it turned out that rule table's wording
   ("in_progress: 10 if it was 0, otherwise unchanged") was ambiguous about
   the specific Done → In Progress case, and once tested live the user
   confirmed the *actual* wanted behavior differs from a pure floor for two
   of the five statuses.
3. **Final, confirmed via explicit per-transition questions** (not
   inferred from a general rule restated in words a second time):

```
Done (100%) -> In Review     => 90%   (reset to the review floor)
Done (100%) -> In Progress   => 10%   (reset exactly like a fresh task)
Done (100%) -> Blocked       => 100%  (unchanged — Blocked carries no
                                        progress signal of its own)
```

Generalized into the full table now in the function's doc comment:

```
To Do        -> always 0%
In Progress  -> 10% if it was 0% OR already 100% (fresh start OR
                regressing from Done are treated the same way);
                otherwise left unchanged (an interrupted/resumed task
                keeps its in-between progress, e.g. 45% stays 45%)
In Review    -> floors at 90%; ALSO snaps down to 90% if it arrives here
                already at/above 100% (same regressing-from-Done case)
Done         -> always 100%
Blocked      -> NEVER changes progress, not even from Done — pure
                "work is stuck" marker, no progress semantics of its own
```

**The key distinction that took three passes to nail down**: `blocked`
truly never touches progress (confirmed twice), while `in_progress` and
`in_review` both got an explicit *additional* reset condition alongside
their existing floor/reset-from-zero logic — `in_review` on `base >= 100`,
`in_progress` on `base >= 90` (widened from `>= 100` in round 4 below,
since In Review → In Progress needed the same reset as Done → In
Progress). A prior in-between value below that threshold (say 45%) is
still left alone in both. Don't re-derive this rule from prose again if it
comes up yet again — ask for the exact number per specific `X → Y`
transition like rounds 3-4 did, since a restated-in-words rule table has
already proven ambiguous twice here.

**Round 4 (2026-07-23, same day)**: In Review → In Progress was reported
still not resetting — confirmed live it should also become 10%, same as
Done → In Progress. Root cause: round 3's `in_progress` condition was
`base == 0 || base >= 100`, so a task arriving from In Review (base 90-99)
didn't qualify. Widened to `base == 0 || base >= 90` — 90+ only ever
originates from In Review or Done, so both now reset In Progress the same
way, while genuine paused/resumed in-progress work below 90% is untouched.

No test file exists for this package yet (checked — `domain/pm/service` has
no `_test.go`); all four revisions were verified via code trace plus
`go build`/`go vet`, never a live write against the shared staging DB, to
avoid mutating real project data for a manual test. A quick manual
smoke-test in the actual UI (Task Detail drawer → Edit → change status
through Done → In Review → In Progress → Blocked, confirm the % shown at
each step) would close the loop the code trace alone can't.
