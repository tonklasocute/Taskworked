# Production Readiness Audit — Taskworked

Date: 2026-08-08
Status: Audit complete — awaiting approval to implement
Auditor scope: Principal Software Architect / Production Readiness review of the
existing codebase (Go/Fiber/GORM backend, React/Vite frontend, Postgres/Redis/MinIO,
Docker Compose). No code was modified during this audit.

## How to read this document

Every finding has: **Severity** (CRITICAL/HIGH/MEDIUM/LOW), **Current
implementation**, **Problem**, **Risk**, **Recommended solution**, **Files**, and
where applicable **DB changes**, **API changes**, **Frontend changes**, **Testing
requirements**. Findings are grouped into thematic sections; each section header
lists which of the 28 requested audit categories it covers. A master severity table
and a P0–P3 roadmap are at the end.

This report reflects the actual current state, not the architecture doc's original
intent — `README.md` and `docs/superpowers/specs/system-architecture-design.md`
are stale in places (e.g. README still says Projects/Tasks/Kanban/etc. are "not yet
built"; they are, in fact, substantially built).

---

## 0. Executive summary

Taskworked is further along than a demo: it has a real modular-monolith backend
(Go/Fiber/GORM), a full React frontend with Kanban/Gantt/Calendar/Reports/Team/
Gamification/AI Assistant features, resource-level authorization that is
**consistently and correctly enforced** (no IDOR found anywhere), clean SQL
(no injection surface), correct password hashing, correct refresh-token rotation,
and broad backend test coverage. This is a materially better starting point than
"prototype."

However, it has **not yet been hardened for production traffic or enterprise
scale**, and it has **no support for the multi-organization/department/team
hierarchy** the target state requires — today it assumes one flat company. The
highest-severity gaps cluster into three groups:

1. **Data-loss / availability blockers**: no backup strategy at all for Postgres or
   MinIO; schema is managed by `AutoMigrate` at every boot with no versioned
   migrations or rollback; no panic recovery (one bad request can crash the whole
   process); no rate limiting anywhere (unlimited login brute-force).
2. **A real, present-day correctness bug**: the Kanban board and project list
   silently cap at 20 items with no pagination in the frontend — tasks past the
   20th simply disappear from view in any active project, today, not at some future
   scale.
3. **Security-sensitive storage/transport choices**: JWTs (access + refresh) live
   in `localStorage` (XSS-exploitable) and the WebSocket handshake carries the
   access token in a URL query string (leaks into logs).

None of this requires a rewrite. The architecture (modular monolith, Clean
Architecture, Repository Pattern) is sound and should be preserved. This is a
hardening and completion pass, not a rebuild — consistent with `PROJECT_RULE.md`
and the standing instruction not to introduce microservices.

---

## 1. Authentication & Session Security
*(covers categories: 5 Authentication, 20 Security)*

### [CRITICAL] JWT access + refresh tokens persisted to `localStorage`
- **Current implementation**: `frontend/src/stores/auth-store.ts:23-34` — Zustand
  `persist` middleware defaults to `localStorage`; both `accessToken` and
  `refreshToken` are part of the persisted state.
- **Problem**: Any XSS (stored task/comment payload, compromised npm dependency,
  malicious browser extension) can read the token pair directly via JS.
- **Risk**: Full, durable account takeover — `localStorage` persists indefinitely,
  unlike a session cookie.
- **Recommended solution**: Move to `httpOnly`, `Secure`, `SameSite=Lax/Strict`
  cookies set by the backend for both tokens; frontend uses `withCredentials: true`
  and drops client-side token storage. If cookie auth isn't immediately feasible,
  at minimum keep the access token in memory only (not persisted) and the refresh
  token in an `httpOnly` cookie.
- **Files**: `frontend/src/stores/auth-store.ts`, `frontend/src/lib/api.ts`,
  `frontend/src/features/auth/api.ts`, `backend/internal/modules/auth/handler.go`,
  `backend/internal/modules/auth/service.go`.
- **API changes**: `/auth/login`, `/auth/refresh`, `/auth/logout` set/clear cookies
  instead of (or in addition to) returning tokens in the JSON body; CSRF protection
  needed (double-submit token or strict `SameSite`) since cookies auto-attach.
- **Testing**: Verify session survives reload but token is not readable via
  `document.cookie`/`localStorage` in devtools; verify logout clears server-side;
  add an XSS-simulation regression test.

### [HIGH] Access token passed as a WebSocket URL query parameter
- **Current implementation**: `backend/internal/realtime/handler.go:29-53` reads
  `c.Query("token")`; `frontend/src/features/tasks/useTaskSocket.ts:17` and
  `frontend/src/features/notifications/useNotificationSocket.ts:16` build the WS URL
  with `?token=<accessToken>`.
- **Problem**: Tokens in URLs land in reverse-proxy/nginx access logs
  (`frontend/nginx.conf` has no scrubbing), browser history, and any intermediary.
- **Risk**: Token leakage via log aggregation independent of TLS.
- **Recommended solution**: Authenticate the WS upgrade via the `httpOnly` cookie
  (browsers auto-attach it) once Finding above lands, or issue a short-lived
  single-use ws-ticket via an authenticated POST just before connecting.
- **Files**: `backend/internal/realtime/handler.go`,
  `frontend/src/features/tasks/useTaskSocket.ts`,
  `frontend/src/features/notifications/useNotificationSocket.ts`.
- **API changes**: WS handshake contract changes (cookie- or ticket-based).
- **Testing**: Confirm socket auth still works after the change; grep logs
  post-fix for absence of token material.

### [MEDIUM] Password-reset flow is defined but not implemented
- **Current implementation**: `backend/internal/modules/auth/dto.go:18-25` defines
  `ForgotPasswordRequest`/`ResetPasswordRequest`; no service method, handler, or
  route exists anywhere (`auth/handler.go:20-36` has no forgot/reset routes).
- **Problem**: Dead DTOs; no self-service recovery path exists.
- **Risk**: Not a vulnerability, but a real operational gap — locked-out users need
  DB-level admin intervention.
- **Recommended solution**: Implement `ForgotPassword`/`ResetPassword` in
  `auth.Service` using Redis-backed single-use reset tokens (mirroring the existing
  refresh-token pattern) and the existing `EmailSender`.
- **Files**: `backend/internal/modules/auth/service.go`, `handler.go`.
- **DB changes**: None (Redis-based tokens).
- **API changes**: New `POST /auth/forgot-password`, `POST /auth/reset-password`;
  update `docs/openapi.yaml`.
- **Testing**: Full flow test including token expiry and single-use enforcement.

### [MEDIUM] JWT parsing doesn't pin the signing algorithm
- **Current implementation**: `backend/internal/modules/auth/tokens.go:50-58,82-97`
  — `ParseWithClaims` keyfunc returns the HMAC secret regardless of the `alg`
  header; no `jwt.WithValidMethods(...)`.
- **Problem/Risk**: Not exploitable today (only HS256 secrets exist anywhere in the
  codebase, and golang-jwt v5 rejects `alg: none` by default), but latent risk if an
  asymmetric-key flow (e.g. future OIDC) is added without revisiting this code.
- **Recommended solution**: Add `jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})`
  to both parse calls as explicit defense-in-depth.
- **Files**: `backend/internal/modules/auth/tokens.go`.
- **Testing**: Unit test rejecting a token signed with a different algorithm.

### [LOW] bcrypt cost factor at library default (10)
- **Current implementation**: `backend/internal/modules/auth/service.go:43` —
  `bcrypt.DefaultCost`.
- **Recommended solution**: Bump to cost 12 (benchmark login latency first) or
  migrate to argon2id. Existing hashes remain valid regardless.
- **Files**: `backend/internal/modules/auth/service.go`.

### What's done well (preserve as-is)
- Bcrypt password storage with correct compare usage.
- JWT secrets fail fast at startup (`mustEnv`, no insecure default fallback).
- Refresh-token rotation + single-active-token-per-user in Redis + true
  "logout everywhere" via one Redis delete — a clean, correct implementation.
- CSRF is a non-issue under the current bearer-token model (no cookies yet).

---

## 2. Authorization, RBAC & Organization Structure
*(covers categories: 6 Authorization, 7 RBAC, 8 Organization structure)*

### [HIGH] No Multi-Organization / Team hierarchy — schema is flat, single-company
- **Current implementation**: `team/model.go` defines only a flat, org-wide
  `Department{ID, Name}` with no parent/hierarchy and no organization scope.
  `project/model.go` has `Project{OwnerID}` + `Member{ProjectID, UserID, Role}`
  (owner/member only). `auth/role.go` defines one global role enum
  (`super_admin`, `admin`, `manager`, `employee`) with no org scoping. There is no
  `Organization` or `Team` entity anywhere — the `team` package is a directory view
  composed from `auth`+`task`+`presence`, not an actual team-membership model. This
  matches the architecture doc's explicit scoping decision ("single company, no
  multi-tenancy") — but the user's target state now requires multi-organization.
- **Problem**: The target requirement "Multi-organization, Department, Team" is not
  representable today at any layer — not schema, not auth, not queries.
- **Risk**: This is the single largest structural gap for the enterprise target.
  Retrofitting later (after real data exists) requires a backfill migration and
  touches nearly every repository's queries and every service's authorization
  checks (`CheckMembership`/`canManage`/`requireMembership` are the backbone of
  every resource-level auth check in the codebase).
- **Recommended solution**: Introduce an `Organization` entity as the top-level
  boundary; scope `Department` under `OrganizationID` with an optional
  `ParentDepartmentID` for hierarchy; introduce a first-class `Team` entity
  (distinct from ad-hoc project membership) with its own membership table, scoped
  to `OrganizationID`/`DepartmentID`; scope `Project` and `User` under
  `OrganizationID`; change `users.email` unique constraint to
  `(organization_id, email)`, `departments.name` to `(organization_id, name)`.
- **Files**: `backend/internal/modules/auth/model.go`, `team/model.go` (new
  `Organization`, `Team` structs), `project/model.go`, and every repository's
  `List*`/`Find*` methods across `auth`, `project`, `task`, `team`, `notification`,
  `gamification` to add org-scoping filters.
- **DB changes**: New `organizations` table; `organization_id` FK on `users`,
  `projects`, `departments`; new `teams` + `team_members` tables;
  `parent_department_id` on `departments`; backfill script assigning existing rows
  to one default organization; drop/recreate the affected unique indexes.
- **API changes**: New CRUD for Organization and Team; existing list endpoints
  (`/projects`, `/team`) gain org-scoping; login/session needs an org-derivation
  step (or org-selection if a user can belong to multiple).
- **Frontend changes**: New Organization/Team management UI; org switcher if
  multi-org membership is supported; Department UI needs a tree view instead of a
  flat dropdown.
- **Testing requirements**: Cross-org isolation tests (user in org A cannot
  read/list org B's data via any endpoint) — this is the most important regression
  suite to add, since a missed `WHERE organization_id = ?` anywhere is a tenant
  data leak; full re-test of every membership/authorization path once scoping
  lands.

### [MEDIUM] `RequireRole` middleware defined but never wired — no route-level RBAC defense-in-depth
- **Current implementation**: `backend/internal/middleware/auth.go:33-46` defines
  `RequireRole(roles ...auth.Role)`; grep across the codebase finds zero call
  sites. Every module instead re-derives the actor's role from
  `httpctx.ActorRole(c)` and checks it inside the **service layer** — verified
  consistently correct across `auth`, `project`, `task`, `team`, `notification`,
  `ai`. This is not currently causing a missing-authorization bug (spot-checked and
  confirmed sound), but it means there's no second layer to catch a future
  regression if one service method forgets its check.
- **Problem**: Dead code that misleads readers into believing route-level RBAC
  exists; no defense-in-depth.
- **Risk**: Low today; structurally fragile for the future.
- **Recommended solution**: Either wire `RequireRole` as a coarse first gate on
  obviously admin-only routes (`PATCH /users/:id/role`, `PATCH /users/:id/department`,
  `POST /departments`, `DELETE /departments/:id`) in addition to the existing
  service-layer check, or delete it if the team commits to service-layer-only
  authorization.
- **Files**: `backend/internal/bootstrap/bootstrap.go`,
  `backend/internal/middleware/auth.go`.
- **Testing**: If adopted, test that non-admin roles get 403 at the middleware
  layer before reaching the service.

### [HIGH] Frontend `ProtectedRoute` has no role-level checks
- **Current implementation**: `frontend/src/routes/ProtectedRoute.tsx:4-7` only
  checks token presence (`accessToken ? <Outlet/> : <Navigate to="/login"/>`).
  Every authenticated route (`/team`, `/gamification`, `/ai-assistant`, project
  routes) is reachable by any logged-in user regardless of role; some pages
  (Team) do gate individual controls client-side, others (Gamification, AI
  Assistant) have no role gating at all.
- **Problem**: No centralized, explicit route-level RBAC intent; inconsistent
  across pages.
- **Risk**: Information disclosure (e.g. org directory/workload data visible to
  roles that may not be intended to see it) and easy to introduce a future bug
  where a sensitive page ships with no role check.
- **Recommended solution**: Introduce a `RequireRole` wrapper component (or a
  `roles` field on route config) so route-level intent is explicit and centralized.
  This is defense-in-depth UX only — backend remains the actual enforcement point
  (confirmed correctly enforced server-side, see below).
- **Files**: `frontend/src/routes/AppRouter.tsx`, new
  `frontend/src/routes/RequireRole.tsx`.
- **Testing**: Manual nav test per role once implemented.

### Confirmed correct (do not rewrite)
- **Resource-level authorization (IDOR) is consistently and correctly enforced**
  server-side across every module checked: `project.Service` verifies
  membership/ownership on every scoped operation; `task.Service` has fine-grained
  per-action ownership checks (author-can-delete-own-comment vs.
  manager-can-delete-any, etc.); `notification.Service.MarkRead` verifies
  ownership before mutating — textbook IDOR guards throughout.
- Privileged mutations (role/department assignment) go through dedicated
  endpoints, not a generic object-update endpoint — no generic-tampering surface.
- The frontend never exposes a form field or endpoint that lets a client set EXP,
  points, or role directly — confirmed on both frontend and backend independently.

---

## 3. Database Schema, Migrations & Indexes
*(covers categories: 3 Database schema, 17 Database indexes)*

### [CRITICAL] No versioned migrations — schema is managed entirely by `AutoMigrate` at boot
- **Current implementation**: `backend/internal/bootstrap/bootstrap.go:104-115`
  (`Migrate`) calls `db.AutoMigrate(...)` against 15 structs, invoked
  unconditionally on every process start (`backend/cmd/api/main.go:19-21`), before
  the server accepts traffic. No migration tool (`golang-migrate`/`goose`/`atlas`)
  is a dependency; no `migrations/` directory exists.
- **Problem**: `AutoMigrate` applies additive DDL with no review step, no
  down-migration, no changelog, and cannot safely rename/drop columns or backfill
  data. `ALTER TABLE ... ADD COLUMN NOT NULL` or new indexes take
  `ACCESS EXCLUSIVE` locks on Postgres, stalling reads/writes on that table for the
  duration — on every deploy.
- **Risk**: A bad model change ships straight to prod schema with no rollback;
  concurrent replicas running `AutoMigrate` simultaneously can race on DDL; routine
  boots become lock/latency incidents as tables grow.
- **Recommended solution**: Adopt a versioned migration tool with up/down file
  pairs, run as an explicit release step (never inside `main()`). Snapshot today's
  AutoMigrate-produced schema as a baseline migration. Keep `AutoMigrate` only for
  local dev, behind a flag.
- **Files**: `backend/internal/bootstrap/bootstrap.go`, `backend/cmd/api/main.go`,
  new `backend/migrations/`, `Makefile`, deploy pipeline.
- **DB changes**: Baseline migration reflecting current schema; every future model
  change ships a matching migration.
- **Testing**: CI step running migrations against a fresh Postgres, asserting
  no drift against models; rollback test for the baseline.

### [HIGH] No foreign key constraints anywhere — referential integrity is application-only
- **Current implementation**: Every relational field (`task.ProjectID`,
  `AssigneeID`, `ParentTaskID`, `MilestoneID`, `Comment.TaskID`,
  `Attachment.TaskID`, `Member.ProjectID/UserID`, `User.DepartmentID`,
  `Notification.UserID`, `Character.UserID`, etc.) is a bare
  `gorm:"type:uuid;index"` — no GORM association, no `REFERENCES`, no
  `OnDelete`. Confirmed via repo-wide grep for `foreignKey|references:|constraint:`.
- **Problem**: Postgres never enforces referential integrity; a task can reference
  a nonexistent project/user/milestone.
- **Risk**: Silent data corruption; deleting a Project/User/Goal/Task leaves
  orphaned children (compounds with soft-delete finding below).
- **Recommended solution**: Add real FK constraints via the migration tool —
  `ON DELETE CASCADE` for child tables (comments, attachments, checklist items,
  watchers, tags, dependencies), `RESTRICT`/`SET NULL` as appropriate for
  `assignee_id`, `department_id`, `milestone_id`.
- **Files**: New migration files.
- **DB changes**: FK constraints on ~20 relationship columns across
  `tasks`, `checklist_items`, `tags`, `dependencies`, `comments`, `attachments`,
  `watchers`, `members`, `goals`, `milestones`, `users`, `notifications`,
  `preferences`, `characters`, `badges`, `mission_progress`.
- **Testing**: Insert-with-dangling-FK rejection tests; cascade-delete behavior
  tests for task children.

### [HIGH] No soft deletes — every delete is permanent, no audit trail, orphans children
- **Current implementation**: Zero uses of `gorm.DeletedAt` anywhere in the
  backend. Every `Delete` method issues a hard `DELETE FROM ...`.
- **Problem**: Deleting a Task permanently destroys it and orphans its comments,
  attachments, checklist items, watchers, tags, dependencies (per the FK finding
  above).
- **Risk**: Accidental or malicious deletion (callable by the reporter or any
  project manager) is unrecoverable without a full DB restore — and there is
  currently no backup strategy either (see §9), so this compounds into total data
  loss risk.
- **Recommended solution**: Add `DeletedAt gorm.DeletedAt \`gorm:"index"\`` to at
  least `Task`, `Project`, `Goal`, `Milestone`, `Comment`, `Attachment`,
  `Department` — GORM's default soft-delete scoping activates automatically once
  the field exists. Add a lightweight audit-log entry specifically for delete
  events (soft-delete alone doesn't record *who* deleted).
- **Files**: `task/model.go`, `project/model.go`, `actionplan/model.go`,
  `team/model.go`; new `audit_logs` table (see §8).
- **DB changes**: `deleted_at TIMESTAMPTZ` (indexed) on the tables above.
- **Testing**: Verify soft-deleted rows excluded from all list/find queries by
  default; verify unique constraints correctly scope around soft-deleted rows.

### [HIGH] No connection pool limits — unbounded `database/sql` defaults
- **Current implementation**: `backend/internal/platform/database/database.go:9-13`
  — `gorm.Open` with no `SetMaxOpenConns`/`SetMaxIdleConns`/`SetConnMaxLifetime`
  anywhere in the codebase.
- **Problem**: Go's default is unbounded open connections; under load or a
  slow-query spike, the process can exceed Postgres's `max_connections`,
  especially with multiple replicas.
- **Risk**: Production outage under moderate concurrent load; connection storms;
  stale connections producing sporadic "bad connection" errors (no
  `ConnMaxLifetime`).
- **Recommended solution**: Set explicit, configurable bounds (e.g.
  `SetMaxOpenConns(25)`, `SetMaxIdleConns(10)`, `SetConnMaxLifetime(30m)`,
  `SetConnMaxIdleTime(5m)`) sized to expected replica count × Postgres capacity.
- **Files**: `backend/internal/platform/database/database.go`,
  `backend/internal/config/config.go` (new env vars).
- **Testing**: Load test with pool limits enforced; verify bounded queueing
  instead of unbounded growth.

### [HIGH] No transactions around multi-step writes — partial failures leave inconsistent state
- **Current implementation**: Only one `.Transaction(` call exists in the entire
  backend (`task/repository.go:132-143`, `SetTags`). Task creation
  (create → set tags → notify), task completion (update status →
  `gamifier.OnTaskCompleted`, error discarded via `_ = ...`), and gamification's
  own EXP/badge/mission writes are each a sequence of independently-committed
  calls with no atomicity.
- **Problem**: A crash or error partway through leaves conceptually-atomic
  operations half-done — e.g. a task marked Done with EXP never awarded, and no
  reconciliation mechanism.
- **Risk**: Silent, cumulative state drift between tasks/gamification/notifications
  since errors are swallowed rather than retried.
- **Recommended solution**: Wrap logically-atomic write groups in
  `db.Transaction(...)` at the repository layer. For cross-module writes (task
  completion → gamification), prefer an outbox pattern: write a `TaskCompleted`
  domain-event row in the same transaction as the task update; a background worker
  processes it into gamification/notification writes with retries — fits the
  existing loosely-coupled holder-interface design better than widening a single
  DB transaction across modules.
- **Files**: `task/repository.go`, `task/service.go` (`Create`, `Update`),
  `gamification/repository.go`, `gamification/service.go`.
- **DB changes**: If outbox adopted: new `domain_events` table
  (`id, type, payload jsonb, created_at, processed_at`).
- **Testing**: Inject a forced failure between steps and assert rollback (or,
  under outbox, eventual retry-to-success).

### [MEDIUM] Missing indexes on frequently-filtered columns
- **Current implementation**: Good baseline coverage exists (`Task.ProjectID`,
  `AssigneeID`, `MilestoneID`, `ParentTaskID`, `User.Email` unique,
  `Notification.UserID`, `Project.OwnerID` all indexed). Gaps: `Task.Status` has no
  index despite being filtered constantly (alone and combined with
  `assignee_id`); `Task.DueDate` has no index despite being sorted on in
  `ListActiveByAssignee`; `Notification(user_id, read)` has no composite/partial
  index despite `CountUnread`/`MarkAllRead` filtering on both.
- **Risk**: Kanban board loads, unread-badge counts, and "my active tasks" degrade
  to sequential scans as data grows.
- **Recommended solution**: `CREATE INDEX idx_tasks_assignee_status ON tasks(assignee_id, status);`,
  `CREATE INDEX idx_tasks_due_date ON tasks(due_date) WHERE due_date IS NOT NULL;`,
  `CREATE INDEX idx_notifications_user_unread ON notifications(user_id) WHERE read = false;`.
- **Files**: New migration file.
- **Testing**: `EXPLAIN ANALYZE` before/after against a seeded 10k+ row dataset.

### [MEDIUM] No optimistic concurrency on Task updates — lost-update race
- **Current implementation**: `task/repository.go:123-125` (`Update`) does a plain
  `db.Save(t)` — full-row overwrite, no version column, no `WHERE updated_at = ?`
  guard.
- **Problem**: Two concurrent `PATCH /tasks/:id` requests (e.g. two users editing
  the same Kanban card, or a status drag racing a field edit) — whichever `Save`
  commits last silently overwrites the other's change, including fields it never
  touched.
- **Risk**: Silent lost updates on collaborative editing — a realistic, frequent
  scenario for a Kanban tool, not a hypothetical.
- **Recommended solution**: Add a `Version int` column; make `Update` conditional
  (`UPDATE tasks SET ..., version = version+1 WHERE id = ? AND version = ?`),
  return 409 on zero rows affected so the client can refetch/retry.
- **Files**: `task/model.go`, `task/repository.go`, `task/service.go`,
  `task/handler.go`.
- **DB changes**: `version integer NOT NULL DEFAULT 0` on `tasks`.
- **API changes**: `PATCH /tasks/:id` can return 409 Conflict.
- **Frontend changes**: `TaskDetailModal.tsx`/Kanban mutation needs 409 handling
  (toast + refetch).
- **Testing**: Concurrency test — two simultaneous updates to the same task,
  assert no field silently lost.

### [MEDIUM] TOCTOU race in task-dependency cycle check
- **Current implementation**: `task/service.go:516-557` (`AddDependency`) reads
  existing dependencies, runs a cycle check in Go, then writes — read and write are
  unsynchronized, no transaction/row lock.
- **Risk**: Two concurrent `AddDependency` calls that are each individually
  cycle-free can combine into a persisted cycle. Degrades gracefully today (Gantt
  view silently omits the critical path rather than crashing) but is a real
  data-integrity gap.
- **Recommended solution**: Wrap read-check-insert in a transaction with
  `SELECT ... FOR UPDATE` (or advisory lock) scoped to the project's dependency
  rows.
- **Files**: `task/repository.go` (`AddDependency`), `task/service.go`.
- **Testing**: Concurrency test firing two conflicting `AddDependency` calls
  simultaneously.

### What's done well (preserve as-is)
- UUID primary keys with server-side `gen_random_uuid()` defaults throughout.
- Composite primary keys correctly used for pure join tables (`Tag`, `Dependency`,
  `Watcher`, `Member`, `Badge`, `MissionProgress`) rather than surrogate keys plus
  a separate unique index.
- `timestamptz` handling is correct by default via the Postgres driver.
- `CompletedAt` kept distinct from `UpdatedAt` — a deliberate, correct design for
  reporting/streak logic.
- Pagination is implemented consistently at the repository layer for `task.List`
  and `project.ListForMember` (the gap is that the frontend doesn't use it — see §5).
- `SetTags`'s transactional delete-then-recreate proves the codebase already knows
  the correct pattern; it just wasn't applied elsewhere.

---

## 4. Backend Resilience, Error Handling & Security Hardening
*(covers categories: 15 Error handling, 16 Validation, 20 Security)*

### [CRITICAL] No panic-recovery middleware — a single handler panic crashes the whole process
- **Current implementation**: `backend/internal/bootstrap/bootstrap.go:213-215`
  registers only `logger.New()` and `cors.New()`. `ErrorHandler` only catches
  returned `error` values, not panics; Fiber does not auto-recover. Handlers do
  unchecked type assertions (`c.Locals("userID").(string)`) that panic on a
  missing/wrong-type local.
- **Risk**: Any single panicking request takes down the whole server process for
  every other in-flight connection — a trivial, unauthenticated DoS.
- **Recommended solution**: `app.Use(recover.New())` registered first, before
  `logger`/`cors`.
- **Files**: `backend/internal/bootstrap/bootstrap.go`.
- **Testing**: A handler that intentionally panics should return 500, not crash
  the process (assert via an e2e test hitting the running app).

### [CRITICAL] No rate limiting anywhere — login/register/refresh fully unthrottled
- **Current implementation**: Zero matches for `limiter`/`ratelimit` anywhere in
  the backend. `auth.Service.Login` does a bcrypt compare with no lockout/backoff.
- **Risk**: Unlimited online brute-force/credential-stuffing against
  `/auth/login`; account-enumeration via `/auth/register` conflict responses;
  refresh-token guessing.
- **Recommended solution**: Add `gofiber/fiber/v2/middleware/limiter` (or a
  Redis-backed limiter, since Redis is already a dependency) globally, with a
  stricter limit scoped to `/auth/login`, `/auth/register`, `/auth/refresh`, and
  the forgot-password endpoints once implemented.
- **Files**: `backend/internal/bootstrap/bootstrap.go`, new
  `internal/middleware/ratelimit.go`.
- **API changes**: New `429 Too Many Requests` responses, documented in
  `docs/openapi.yaml`.
- **Testing**: Hammer `/auth/login` past the threshold, assert 429.

### [HIGH] No security headers (CSP, X-Frame-Options, HSTS, X-Content-Type-Options)
- **Current implementation**: Only `cors.New()` and `logger.New()` registered; no
  `helmet` middleware anywhere.
- **Risk**: Clickjacking, MIME-sniffing, no HSTS enforcement once behind TLS, no
  baseline CSP for the Swagger UI (which itself pulls a CDN script — see §9).
- **Recommended solution**: `app.Use(helmet.New())`, tuned for a JSON API.
- **Files**: `backend/internal/bootstrap/bootstrap.go`.

### [HIGH] Overly permissive CORS — no origin allowlist
- **Current implementation**: `bootstrap.go:215` — `cors.New()` with zero config;
  Fiber v2 defaults to `AllowOrigins: "*"`. `AllowCredentials: false` limits
  immediate exploitability, but there's no environment-based allowlist at all.
- **Recommended solution**: Add `CORS_ALLOWED_ORIGINS` env var (comma-separated),
  load into `config.Config`, pass to `cors.New(cors.Config{AllowOrigins: ...})`.
- **Files**: `backend/internal/config/config.go`, `backend/internal/bootstrap/bootstrap.go`,
  `.env.example`.
- **Testing**: Preflight `OPTIONS` from a non-allowlisted origin is rejected.

### [HIGH] No graceful shutdown — deferred cleanup is unreachable dead code
- **Current implementation**: `backend/cmd/api/main.go:29` —
  `log.Fatal(app.Fiber.Listen(...))`. No `signal.Notify`/SIGTERM handling. Because
  `log.Fatal` calls `os.Exit` internally, the `defer app.StopSubscriber()` /
  `defer app.CronScheduler.Stop()` at lines 26-27 **never execute on any exit
  path** — they are dead code today.
- **Risk**: On `docker stop`/SIGTERM/rolling deploy, in-flight requests are
  hard-killed; the Redis pub/sub subscriber and cron scheduler are never cleanly
  stopped; zero-downtime deploys are impossible as-is.
- **Recommended solution**: Standard graceful-shutdown pattern —
  `signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)` in a goroutine, then
  `app.Fiber.ShutdownWithContext(ctx)` with a timeout, then run the existing
  cleanup calls in the shutdown path (not as unreachable defers).
- **Files**: `backend/cmd/api/main.go`.
- **Testing**: Send SIGTERM under load; verify in-flight requests complete and
  cleanup methods are actually invoked.

### [MEDIUM] No Fiber Read/Write/Idle timeouts
- **Current implementation**: `bootstrap.go:206-213` sets `BodyLimit` and
  `ErrorHandler` only — no `ReadTimeout`/`WriteTimeout`/`IdleTimeout`.
- **Risk**: Slowloris-style slow-client connections can hold worker capacity
  indefinitely.
- **Recommended solution**: Set explicit timeouts (e.g. 10s/10s/60s, with a longer
  write allowance carved out for the attachment-upload path).
- **Files**: `backend/internal/bootstrap/bootstrap.go`.

### [MEDIUM] No structured logging or request-ID correlation
- **Current implementation**: Fiber's default text `logger.New()` for access logs;
  application code uses plain `log.Printf`/`log.Fatalf`. Zero use of
  `zerolog`/`zap`/`log/slog` anywhere. No request-ID middleware.
- **Risk**: Cannot correlate an access-log line with the app-level logs (GORM
  warnings, notification-failure logs) it produced; hard to ship to a log
  aggregator with structured queries.
- **Recommended solution**: Adopt `log/slog` (stdlib, zero new dependency, fits Go
  1.25) with JSON output; add `gofiber/fiber/v2/middleware/requestid`; thread the
  request ID through every log line; echo `X-Request-Id` in responses.
- **Files**: `backend/internal/bootstrap/bootstrap.go`, every `log.` call site
  across `backend/internal/**`.

### [LOW] GORM logs full SQL including bind values at Warn level
- **Current implementation**: `backend/internal/platform/database/database.go:9-13`
  — `logger.Default.LogMode(logger.Warn)` still logs full SQL with interpolated
  values for slow/erroring queries, which may include emails or other PII.
- **Recommended solution**: Route through structured logging once adopted;
  consider redacting bind parameters.
- **Files**: `backend/internal/platform/database/database.go`.

### [LOW] MinIO credentials silently default to empty string
- **Current implementation**: `config.go:56-57` uses `env(..., "")` for MinIO
  credentials, unlike JWT secrets which use `mustEnv` and panic if missing.
- **Risk**: Low — the code already degrades gracefully (documented in
  `bootstrap.go:132-134`), just an inconsistency in secret-handling discipline.
- **Recommended solution**: Normalize — either `mustEnv` if object storage is
  required in production, or document the optional/degraded behavior explicitly.
- **Files**: `backend/internal/config/config.go`.

### Confirmed correct (do not rewrite)
- Error responses never leak internals — `httpctx.WriteErr` and the global
  `ErrorHandler` both map unknown errors to a generic message, never raw Go
  errors/SQL errors/stack traces. Consistently applied everywhere.
- `go-playground/validator` is wired into every handler that accepts a body — no
  handler found parsing without validating.
- SQL injection surface is clean — every repository query is parameterized;
  no string-concatenated SQL found anywhere.
- CSRF is correctly not implemented, since bearer-token auth has no cookies to
  exploit (revisit if/when the cookie-based auth change above lands).

---

## 5. Task Management, Kanban, Gantt & Concurrency
*(covers categories: 9 Task management, 19 Concurrency, 4 API design)*

### [CRITICAL] Kanban board / project list silently truncate at 20 items — frontend never paginates
- **Current implementation**: `task/handler.go:97-121` defaults `page_size=20`
  (clamped to `[1,100]` in `task/service.go:267-272`). The Kanban board
  (`frontend/src/features/tasks/KanbanBoard.tsx:35-39`, `api.ts:16-20`) calls
  `listTasks(projectId)` with **no page/page_size parameter**, then distributes
  `tasks.items` into 7 status columns by client-side filter. `ProjectsPage.tsx`
  does the same for `listProjects()`.
- **Problem**: Any project with more than 20 tasks has tasks silently vanish from
  the Kanban board — no error, no "load more," no total count shown anywhere. This
  is a present-day correctness bug in any moderately active project, not a future
  scale concern.
- **Risk**: Users lose visibility into real work; data appears to have
  disappeared — a severe trust/reliability issue for a task-management product.
- **Recommended solution**: Short-term, add a membership-checked "board view"
  endpoint that returns everything in a project unpaginated (bounded by a sane
  ceiling, e.g. 1,000) for Kanban's must-show-every-card nature; longer-term,
  implement true cursor/keyset pagination with visible "load more"/virtualization
  in the Kanban and Projects list UIs.
- **Files**: `frontend/src/features/tasks/api.ts`,
  `frontend/src/features/tasks/KanbanBoard.tsx`,
  `frontend/src/features/projects/api.ts`,
  `frontend/src/features/projects/ProjectsPage.tsx`; possibly
  `backend/internal/modules/task/handler.go`/`service.go` for a new endpoint.
- **API changes**: New/adjusted "give me everything in this project" endpoint, or
  the frontend loops pages.
- **Testing**: Integration test with 100+ tasks in one project verifying all
  appear across all Kanban columns; regression test for the projects list beyond
  20 items.

### [HIGH] Report/Gantt/ActionPlan load the entire task set into memory; N+1 tag queries
- **Current implementation**: `report/service.go:36-155` calls
  `taskSvc.ListAllResponses(...)` (unfiltered `SELECT * FROM tasks WHERE project_id = ?`,
  no LIMIT), then computes all summary/performance/productivity aggregation in Go
  loops. Separately, `task/service.go:279-286,584-591,681-688` (`List`,
  `ListAllResponses`, `GetGanttView`) each loop over fetched tasks calling
  `s.repo.Tags(ctx, tasks[i].ID)` individually — one extra query per task.
- **Risk**: Report/CSV-export latency and memory scale linearly with total
  historical task count, not the reporting period; N+1 tag fetches compound this
  and risk connection-pool exhaustion under concurrent report requests.
- **Recommended solution**: Push `Summary`/`AssigneePerformance`/`ProductivityPoint`
  into SQL aggregate queries (`GROUP BY`/`COUNT`/`date_trunc`) scoped by period;
  stream CSV export from a cursor rather than materializing everything. Batch tag
  fetches with a single `WHERE task_id IN (...)` query (the codebase already uses
  this batching pattern elsewhere — `enrichUserNames`, `task/service.go:1018-1031`
  — it just wasn't applied to tags).
- **Files**: `report/service.go`, `task/repository.go` (new aggregate + batched-tag
  methods), `task/service.go`.
- **DB changes**: Indexes to support aggregates efficiently — `(project_id, status, due_date)`,
  `(project_id, completed_at)`.
- **Testing**: Load test with a large synthetic task set for bounded time/memory;
  unit tests for new aggregate queries against real Postgres.

### [MEDIUM] Team directory issues one workload/presence query per user, unbounded
- **Current implementation**: `team/service.go:68-114` (`GetDirectory`) calls
  `authSvc.ListUsers(ctx)` (all users, unpaginated) then, per user, a separate
  `GetWorkload` DB query and a separate Redis presence lookup — N+2 round trips for
  an org of N users.
- **Recommended solution**: Batch workload with one `GROUP BY assignee_id` query;
  batch presence lookups with a single Redis `MGET`/pipeline; add pagination/search
  to the directory endpoint.
- **Files**: `team/service.go`, `team/repository.go`, `task/repository.go`.
- **API changes**: `GET /team` gains `page`/`page_size`/`search`.
- **Frontend changes**: `TeamPage.tsx`, `team/api.ts` need pagination support.

### [MEDIUM] No file-type allow-list or malware scanning on task attachments
- **Current implementation**: `task/handler.go:443-482` checks only size (20MB
  cap); no content-type/extension validation, no server-side content sniffing
  (client-supplied `Content-Type` is trusted and passed straight to MinIO).
- **Risk**: Any file type (executables, HTML with embedded scripts) can be
  uploaded and later downloaded by any project member via presigned URL — a
  content-safety gap.
- **Recommended solution**: Configurable allow/deny-list of extensions/MIME types;
  server-side content sniffing (`http.DetectContentType`) instead of trusting the
  client header; force `Content-Disposition: attachment` on all presigned URLs
  regardless of type so nothing renders inline.
- **Files**: `task/handler.go`, `task/service.go` (`UploadAttachment`),
  `backend/internal/platform/storage/minio.go`.
- **Frontend changes**: Surface rejected-type errors; pre-filter file picker.

### [MEDIUM] No recurring-task support (feature gap)
- **Current implementation**: No recurrence fields on `Task`, no occurrence
  generation anywhere.
- **Recommended solution**: Add a `RecurrenceRule` on `Task` plus a scheduler
  (the codebase already runs `robfig/cron` — extend it) to materialize the next
  occurrence on completion or schedule.
- **Files**: `task/model.go`, `dto.go`, `service.go`, `repository.go`,
  `bootstrap.go` (new cron job).
- **DB changes**: New recurrence columns/table.
- **Frontend changes**: UI for setting/editing recurrence and indicating series
  membership.

### [LOW] No intra-column Kanban ordering; no workflow state-machine on status; unsanitized filename in object key; no sort option / non-indexable search
- These four are grouped as lower-priority completeness items:
  - No `Position`/`Order` field on `Task` — cards can move between columns but not
    be manually reordered within one. Add a fractional-index `Position` column +
    a reorder endpoint if required (pairs with the optimistic-locking fix above,
    since reordering is a genuine concurrent-write surface).
  - Any status→any status transition is currently legal
    (`task/dto.go:23`, `oneof=backlog todo doing review testing done blocked`)
    with no workflow gates — likely intentional for a flexible tool, flagged per
    the audit's explicit ask, not treated as a defect.
  - `task/service.go:814-816` builds the MinIO object key directly from the
    client-supplied filename with no sanitization (no traversal risk since a fresh
    UUID guarantees key uniqueness, but download-filename spoofing/metadata
    pollution risk remains) — sanitize or store only a UUID key with the original
    name kept as DB metadata.
  - `List` hardcodes `ORDER BY created_at DESC` with no client-controllable sort,
    and search uses a leading-wildcard `ILIKE '%term%'` that can't use a B-tree
    index — add an allow-listed `sort` param and a trigram/full-text index for
    search as the dataset grows.
- **Files**: `task/model.go`, `repository.go`, `service.go`, `handler.go`,
  `frontend/src/features/tasks/KanbanBoard.tsx`; `project/repository.go` (search).

### Confirmed correct (do not rewrite)
- Resource-level authorization has no IDOR gaps anywhere in task/project/actionplan.
- Critical Path Method implementation (`task/critical_path.go`) is correct and
  well tested — proper forward/backward pass, slack computation, Kahn's-algorithm
  cycle detection, deterministic tie-breaking; covered by tests for linear chains,
  diamonds, disconnected components, milestones, dangling references, cycles.
- Dependency-cycle prevention at write time (separate from the TOCTOU race noted
  in §3) is correctly implemented and tested.
- Attachment downloads use short-TTL (15 min) presigned URLs with membership
  gating before a URL is even issued — correct pattern.
- Business-rule validation (start ≤ due date, assignee-must-be-project-member,
  parent-task-same-project, cross-project-dependency rejection) is present and
  tested.
- Realtime WebSocket hub is simply and correctly synchronized (`sync.RWMutex`,
  per-topic client sets, isolated per-client broadcast failures), and Redis
  pub/sub as the fan-out layer means it's already horizontally-scalable — a good
  early decision.

---

## 6. Notification System & LINE Integration
*(covers categories: 10 Notification system, 11 LINE integration)*

### [HIGH] Notification delivery (email + LINE) is fully synchronous in the HTTP request path
- **Current implementation**: `task.Service.Update`/`AddComment`/`UploadAttachment`
  call `notifier.NotifyAssignment(...)` inline, in the same goroutine serving the
  request, before responding. `notification/sender.go:38` uses
  `http.DefaultClient` for LINE with **no timeout configured**.
- **Risk**: A slow/hung LINE API or SMTP server directly degrades or blocks task
  write requests; a downed external service can block a request indefinitely.
- **Recommended solution**: Move channel delivery off the request path — publish a
  lightweight job to a Redis-backed queue (e.g. `asynq`, or a simple Redis
  Stream/List worker) from `notify()`, return immediately after the cheap DB
  create + in-app broadcast, have a background worker handle email/LINE delivery
  with retry. At minimum, give the LINE client an explicit timeout immediately.
- **Files**: `notification/sender.go`, `notification/service.go`, new
  `internal/platform/queue` package, `bootstrap.go`.
- **DB changes**: New `notification_deliveries` table (see next finding).

### [HIGH] No retry, dead-letter, or delivery-status tracking — this is the "Notification Queue" requirement, and it doesn't exist yet
- **Current implementation**: `notification/service.go:204-217` — send failures
  are only `log.Printf`'d and swallowed; the function still returns success.
  `Notification` has no `delivery_status`/`retry_count`/`last_error` column.
- **Risk**: Users silently miss assignment/comment notifications during any
  transient outage, with zero visibility and no retry.
- **Recommended solution**: Add a `notification_deliveries` table tracking
  per-channel status/attempts/last-error; retry with exponential backoff (e.g. 3
  attempts) via the async worker above; mark `dead_letter` after max attempts,
  optionally surfaced in an admin view.
- **DB changes**: `notification_deliveries(id, notification_id, channel, status, attempts, last_error, created_at, updated_at)`.

### [MEDIUM] LINE integration: no rate-limit/backoff handling; token stored in plaintext; possible service deprecation
- **Current implementation**: `notification/sender.go:33-59` has no
  429/`Retry-After` handling; `Preference.LineNotifyToken` is a plain-text DB
  column with no encryption at rest. LINE Notify (`notify-api.line.me`) has been
  reported as deprecated/sunset by LINE for new integrations — needs verification
  against current LINE developer docs, since the entire channel is at risk if so.
- **Recommended solution**: Verify LINE Notify's current status and migrate to the
  LINE Messaging API (push message + channel access token) if Notify is
  deprecated; add 429/backoff handling; encrypt `LineNotifyToken` at rest
  (application-level AES-GCM or `pgcrypto`).
- **Files**: `notification/sender.go`, `model.go`, `repository.go`, `service.go`.
- **DB changes**: Migration to re-encrypt/backfill the token column.

### [MEDIUM] Cron digest jobs have no distributed lock — will duplicate-send once scaled beyond one replica
- **Current implementation**: `bootstrap.go:195-201` — plain `cron.New()`, no
  `cron.WithChain(cron.SkipIfStillRunning(...))`, no distributed lock, single
  process only.
- **Risk**: If the API is ever horizontally scaled (a normal production step),
  every replica independently fires each digest job at the scheduled time — every
  user gets N duplicate daily/EOD/weekly/monthly notifications. This is a
  near-certain issue the moment the service scales, not hypothetical.
- **Recommended solution**: Redis `SETNX`-based distributed lock around each
  digest job (Redis is already in the stack), or run cron in a single dedicated
  worker replica separate from API replicas. Add `SkipIfStillRunning` regardless
  as a cheap safety net.
- **Files**: `backend/internal/bootstrap/bootstrap.go`, new lock helper using the
  existing Redis client.

### [MEDIUM] Digest queries are unbounded and sequential — no batching
- **Current implementation**: `notification/service.go:289-302` (`forEachUser`)
  loads the entire user table in one call, then loops sequentially with 1-2 DB
  round trips per user, no concurrency limit, no context deadline.
- **Recommended solution**: Paginate `ListUsers`; process in bounded-concurrency
  batches (e.g. `errgroup` with a semaphore of 10-20); add a per-job time budget.
- **Files**: `notification/service.go`, `auth/service.go` (`ListUsers` pagination).

### [LOW] `GetLeaderboard` full-table scan with O(n²) sort, no pagination/caching
- **Current implementation**: `gamification/repository.go:50-54` loads the entire
  `characters` table with no LIMIT; `sortDepartmentsByEXPDesc` is a hand-rolled
  O(n²) insertion sort.
- **Recommended solution**: Add `LIMIT`/pagination, an index on `characters(exp DESC)`,
  replace the sort with `sort.Slice`, cache the response in Redis with a short TTL
  (30-60s — real-time freshness isn't critical for a leaderboard).

### [LOW] Notification/gamification errors from task actions are silently discarded with no metric
- **Current implementation**: `task/service.go:371,381,735,835` — `_ = ...`
  discards notifier/gamifier errors. This is actually the *correct* choice for not
  failing the primary task-mutation request, but combined with the findings above
  it means failures are currently invisible — no logging, no metric.
- **Recommended solution**: At minimum, log with structured fields (task ID, user
  ID, error) and emit a counter metric so a spike is alertable, independent of the
  deeper queue/retry fix above.

### Confirmed correct (do not rewrite)
- Cron schedule matches the architecture doc exactly (08:00 daily, 18:00 EOD,
  Monday 07:30 weekly, 1st-of-month 07:00 monthly) and is genuinely implemented,
  not just documented intent.
- Digest jobs correctly skip sending empty digests — a thoughtful touch avoiding
  inbox/LINE spam.
- The `task.Notifier`/`task.Gamifier` interface-based decoupling is clean Go
  architecture and correctly avoids import cycles — the fix needed is *runtime*
  behavior (sync → async), not the interface design.

---

## 7. Gamification
*(covers category: 12 Gamification)*

### [HIGH] EXP read-modify-write is not atomic — race condition on concurrent completions
- **Current implementation**: `gamification/service.go:46-107` (`OnTaskCompleted`)
  does a plain `SELECT` (`repository.go:34-44`, no `FOR UPDATE`), mutates
  `EXP`/`Level`/`TotalCompleted`/`CurrentStreak` in Go memory, then a plain
  `Save`/`UPDATE` — no transaction spanning the read+write, no version column.
- **Problem**: Two goroutines completing tasks for the same user near-simultaneously
  (two browser tabs, two rapid API calls) both read the same stale `EXP` base
  value; the second `Save` silently overwrites the first — one award is lost.
- **Risk**: Silent EXP/streak/badge loss under real concurrent load — directly
  undermines leaderboard/level trust, and this is a live bug pattern, not
  theoretical.
- **Recommended solution**: Wrap `OnTaskCompleted` (and `bumpMission`, same issue)
  in a transaction using `SELECT ... FOR UPDATE` on the character row, or push the
  arithmetic itself into atomic SQL (`UPDATE characters SET exp = exp + ?, total_completed = total_completed + 1 ... WHERE user_id = ?`),
  reserving Go logic only for decisions that need the pre-image (streak-date,
  badge thresholds) inside the same locked transaction.
- **Files**: `gamification/repository.go` (locking/transactional method),
  `gamification/service.go` (`OnTaskCompleted`, `bumpMission`).
- **DB changes**: Optional `version` column if optimistic locking is preferred
  over pessimistic row locks.
- **Testing**: Concurrency test firing N goroutines completing tasks for the same
  user simultaneously against a real (or transactional-test) Postgres, asserting
  the final EXP/TotalCompleted equals the expected sum — the current in-memory
  fake repository in tests cannot catch this class of bug.

### [HIGH] EXP farming exploit — no once-per-task guard on completion rewards
- **Current implementation**: `task/service.go:310-319` sets `justCompleted = true`
  any time a PATCH sets `Status: Done` while the stored status isn't already
  `Done`. Toggling away from Done clears `CompletedAt` with **nothing preventing
  re-completion** later, which re-triggers the EXP/badge/streak award.
- **Problem**: A user can create a trivial task, mark it done (+EXP), mark it
  todo (free), mark it done again (+EXP again) — indefinitely. No per-task
  "already rewarded" flag, no daily cap, no cooldown. `TotalCompleted` (which
  gates the fast_worker/hundred_tasks/legend badges) is farmable the same way.
- **Risk**: Trivial, high-volume EXP/leaderboard/badge farming — directly
  undermines gamification integrity, which the requirements explicitly flagged as
  a must-check.
- **Recommended solution**: Add an `ExpAwarded bool` (or `RewardedAt *time.Time`)
  column on `Task` so completion rewards — and `TotalCompleted`/badge/streak
  logic — fire **at most once per task, ever**, regardless of status oscillation.
  Un-completing a previously-rewarded task should not retroactively decrement
  (to avoid a different exploit) but should permanently mark it counted.
  Optionally add a daily EXP cap as defense-in-depth.
- **Files**: `task/model.go` (new column), `task/service.go` (`justCompleted`
  logic and the completion-trigger call site), `gamification/service.go`
  (idempotency check as a second layer).
- **DB changes**: `ALTER TABLE tasks ADD COLUMN exp_awarded boolean NOT NULL DEFAULT false;`
- **Testing**: Test completing → un-completing → re-completing the same task
  grants EXP only once; test badge thresholds aren't reachable by oscillating one
  task.

### Confirmed correct (do not rewrite) — this satisfies an explicit security requirement
- **No endpoint anywhere accepts a client-supplied EXP/points value.**
  `gamification/handler.go` exposes only read-only `GET /gamification/profile` and
  `GET /gamification/leaderboard`. Every write path to `Character.EXP` was traced
  and confirmed to originate only from server-side rule-table constants
  (`+20`/`+50`/`+100`/`-20` in `OnTaskCompleted`, mission-reward constants in
  `model.go`), never from request input. This is exactly the "never allow frontend
  to assign EXP directly" requirement, and it is currently satisfied — the two
  findings above are about *integrity of the trigger*, not about a
  client-writable field. Keep this invariant true as new endpoints are added.

---

## 8. AI Assistant
*(covers category: as part of core modules; security/cost control per explicit requirements)*

### [HIGH] No per-user/per-org rate limiting or cost control on Claude API calls
- **Current implementation**: All 9 AI endpoints (generate-tasks, daily/weekly
  summary, estimate-duration, suggest-priority, suggest-assignee, predict-late,
  productivity, meeting-summary) have no rate limiting — confirmed via a
  backend-wide grep for `limiter`/`RateLimit` (zero matches anywhere, not just in
  AI). Every authenticated user can call any endpoint unlimited times; each call
  is a real, billed Claude API request.
- **Risk**: Unbounded, direct financial exposure from a single user, buggy client,
  or scripted loop — explicitly flagged as a must-check in the requirements.
- **Recommended solution**: Per-user and per-org rate limiting (Redis token-bucket,
  e.g. N requests/minute and/or M requests/day), returning 429 when exceeded.
  Capture `usage` from Claude API responses (currently not even parsed —
  `ai/client.go`'s response struct omits it) to enable per-org monthly budget caps
  as a hard stop.
- **Files**: `ai/client.go` (capture usage), new rate-limit middleware,
  `bootstrap.go` (AI route group).
- **DB changes**: Optional `ai_usage` table for auditing
  (`user_id, org_id, endpoint, tokens_in, tokens_out, cost_estimate, created_at`).
- **API changes**: 429 response shape for rate-limited AI calls.

### [MEDIUM] AI calls are fully synchronous with only a coarse 60s timeout, no circuit breaker
- **Current implementation**: `ai/client.go:48` — `http.Client{Timeout: 60s}`;
  every service method blocks the HTTP handler for up to 60s.
- **Risk**: Poor UX during Claude slowness (60s hangs feel broken); cascading
  resource pressure on the API server during an Anthropic outage (many goroutines
  parked for up to 60s each).
- **Recommended solution**: Lower the synchronous timeout to something
  UX-appropriate (20-30s); add a circuit breaker (e.g. `sony/gobreaker`) so
  repeated failures short-circuit instead of repeatedly waiting out the full
  timeout; consider moving the heaviest calls (generate-tasks, weekly-summary,
  productivity) to an async job + polling/WebSocket pattern, keeping only cheap
  calls synchronous.
- **Files**: `ai/client.go`, `ai/service.go`, `ai/handler.go`.

### [LOW-MEDIUM] Prompt injection surface — user text interpolated directly into prompts
- **Current implementation**: `ai/service.go` builds prompts via direct
  `fmt.Sprintf`/string concatenation of user-controlled fields (task
  descriptions, meeting notes up to 20,000 chars) with no delimiter/escaping.
- **Risk assessment**: Meaningfully bounded today — every AI response is
  constrained to a structured JSON schema (`additionalProperties: false`) and is
  returned to the client purely as a *suggestion*; no AI output currently
  auto-triggers a DB write or further action. Blast radius today is a misleading
  suggestion shown to a human, not an autonomous state change.
- **Recommended solution**: Wrap interpolated content with explicit
  data-not-instructions delimiters as low-cost hardening now; if any future
  feature lets AI output auto-apply state changes without human confirmation, add
  strict output allow-listing before that lands.
- **Files**: `ai/service.go` (prompt construction helpers).

### Confirmed correct (do not rewrite)
- `ANTHROPIC_API_KEY` is loaded server-side only from env, used only as an
  outbound header, and never appears in any response DTO — confirmed on both
  backend and frontend independently. No secrets are bundled into the frontend
  build.
- Structured JSON-schema output (`additionalProperties: false`) meaningfully
  constrains what the model can return, limiting both misuse and the practical
  impact of prompt injection.

---

## 9. File Storage
*(covers category: 13 File storage — see also §5 for attachment-specific findings)*

Findings on upload validation and object-key sanitization are covered in §5
(Task Management) since they're specific to the task-attachment flow, the only
current consumer of MinIO. General storage-layer notes:

- Presigned-URL pattern (15-minute TTL, membership-gated before issuance) is
  correctly implemented — **preserve as-is**.
- MinIO connection failures degrade gracefully at boot (logged as a warning, not
  fatal) rather than blocking the whole API from starting — a good defensive
  pattern already in the codebase (`bootstrap.go:132-140`).
- No backup/replication for MinIO data — see §11 (Backup/Recovery), CRITICAL.

---

## 10. Logging, Monitoring & Observability
*(covers categories: 14 Logging, 24 Monitoring)*

### [MEDIUM] `/health` is a liveness stub, not a real readiness check
- **Current implementation**: `bootstrap.go:217-219` unconditionally returns
  `200 {"status":"ok"}` regardless of DB/Redis/MinIO state. No separate
  `/ready`/`/live` endpoint exists.
- **Risk**: An orchestrator can't distinguish "process is up" from "process can
  actually serve traffic" — a container can be marked healthy while every real
  request 500s (e.g. DB pool exhausted).
- **Recommended solution**: Split into `/healthz` (liveness) and `/readyz`
  (pings DB via `PingContext` and Redis via `Ping`, returns 503 on failure). Wire
  a Docker Compose `healthcheck:` block on the `api` service against `/readyz`
  (currently `api` has no healthcheck at all, unlike `postgres`/`redis`).
- **Files**: `backend/internal/bootstrap/bootstrap.go`, `docker-compose.yml`.

### [MEDIUM] No metrics/Prometheus instrumentation
- **Current implementation**: Zero matches for `prometheus`/`/metrics` anywhere.
- **Recommended solution**: Add a Fiber Prometheus middleware exposing
  `/metrics`; instrument DB pool stats, Redis pool stats, and business counters
  (task creates, notification sends, AI calls). Stand up Prometheus + Grafana in
  a production compose overlay.
- **Files**: `bootstrap.go`, `docker-compose.prod.yml` (new).

### [MEDIUM] No error tracking / APM
- **Current implementation**: Zero matches for `sentry`/`otel`/`opentelemetry`/`datadog`.
- **Recommended solution**: Integrate Sentry or OpenTelemetry for panic/error
  capture with stack traces and request context (pairs directly with the
  panic-recovery middleware in §4 — recovery should report to this, not just
  return 500).
- **Files**: `bootstrap.go`.

### (Duplicate cross-reference) No structured logging / request-ID correlation
- Covered fully in §4; also blocks meaningful log-based observability at scale.

### [LOW] No request ID / correlation ID middleware
- Covered as part of the structured-logging finding in §4 — called out
  separately here because it's independently cheap to add
  (`gofiber/fiber/v2/middleware/requestid`) even before a full logging migration.

### Confirmed correct (do not rewrite)
- Fiber's access-log middleware and MinIO's graceful startup-degradation logging
  are reasonable as a starting point — the gap is structure and correlation, not
  a fundamental redesign.

---

## 11. Backup, Recovery & Data Durability
*(covers category: 25 Backup/Recovery)*

### [CRITICAL] No backup/recovery strategy for Postgres or MinIO at all
- **Current implementation**: Repo-wide search for `pg_dump`/`pg_restore`/`backup`
  returns zero matches anywhere. `docker-compose.yml` defines `postgres_data`,
  `redis_data`, `minio_data` as plain named Docker volumes with no snapshot,
  replication, or export mechanism attached.
- **Problem**: Data durability depends entirely on a single Docker volume on a
  single host surviving forever.
- **Risk**: This is the highest-severity finding in the entire audit. Host disk
  failure, an accidental `docker volume rm`, a bad migration, or
  `docker compose down -v` destroys **all** production data — every task,
  project, user, and attachment — with zero recovery path. Compounds directly
  with the no-soft-deletes finding (§3): there isn't even an in-app undo, let
  alone an infrastructure-level one.
- **Recommended solution**: At minimum, scheduled `pg_dump` (cron
  container/host cron) writing to off-host storage, with a documented,
  periodically-tested restore runbook. For MinIO, enable bucket versioning and/or
  scheduled `mc mirror` to an off-host target. Given the single-VPS deployment
  model, even a simple nightly `pg_dump | gzip | upload-offsite` script is a huge
  improvement over the current nothing.
- **Files**: New `ops/backup-postgres.sh` (or equivalent), new compose
  service/host cron entry, restore runbook documentation.
- **Testing**: Scheduled dump produces a valid, restorable archive; perform an
  actual restore-into-scratch-database drill; verify off-host copy is retrievable.

---

## 12. Audit Logging
*(covers category: 26 Audit logging)*

### [HIGH] No structured audit log for sensitive actions
- **Current implementation**: Zero matches for `audit` anywhere in the backend.
  Sensitive mutations (`UpdateRole`, `UpdateDepartment`, project/task deletion,
  member removal) perform the action and return, with no persisted record of who
  changed what and when beyond GORM's generic `UpdatedAt`.
- **Risk**: No forensic trail for privilege escalation, data-loss disputes, or
  compliance requirements (SOC2/ISO27001-style controls typically require this for
  role/permission changes) — and this is an explicit target-state requirement.
- **Recommended solution**: Add an `audit_logs` table
  (`actor_id, action, target_type, target_id, metadata jsonb, created_at`) and a
  small `audit.Service.Record(...)` called from the sensitive service methods —
  at minimum role changes, project/task deletion, and member removal. Pairs
  naturally with the soft-delete work in §3 (record the actor on every delete).
- **Files**: New `internal/modules/audit/*`, call sites added in `auth/service.go`,
  `project/service.go`, `task/service.go`, `team/service.go`.
- **DB changes**: New `audit_logs` table.
- **API changes**: New `GET /audit-log` admin endpoint.
- **Testing**: Unit tests asserting an audit row is written on role change and on
  deletion.

---

## 13. Frontend Architecture, Validation & Error Handling
*(covers categories: 1 Frontend architecture, 16 Validation, 15 Error handling)*

### [HIGH] No global React error boundary
- **Current implementation**: `frontend/src/main.tsx`/`App.tsx` tree has no
  `ErrorBoundary` anywhere (zero matches for `ErrorBoundary`/`componentDidCatch`).
- **Risk**: Any uncaught render-time exception (malformed API response, a
  third-party chart/calendar library throwing) unmounts the entire React tree —
  blank white screen, no recovery, nothing reported anywhere.
- **Recommended solution**: Top-level `ErrorBoundary` (class component or
  `react-error-boundary`) wrapping `AppRouter`, with a fallback UI and reload
  action; add route-level `errorElement` on `createBrowserRouter` entries
  (React Router v7 supports this natively; none of the current routes use it) so
  a crash in one tab (Gantt/Calendar) doesn't kill the whole shell.
- **Files**: `frontend/src/App.tsx`, new `frontend/src/components/ErrorBoundary.tsx`,
  `frontend/src/routes/AppRouter.tsx`.

### [MEDIUM] No form validation library — ad-hoc HTML attributes only
- **Current implementation**: Every form (login, register, project/task create,
  action plan) relies solely on native `required`/`minLength`/`type="email"` plus
  manual `.trim()` checks. No `zod`/`yup`/`react-hook-form` anywhere.
- **Risk**: No shared validation contract with the backend; generic, guessed
  error messages instead of field-specific ones (`extractErrorMessage` in
  `App.tsx` only reads a flat string, can't render field-level errors even if the
  backend sends structured ones).
- **Recommended solution**: Adopt `react-hook-form` + `zod` for the real forms,
  with schemas mirroring backend validation; extend the error normalizer to
  detect structured validation-error shapes and route them to the right field.
- **Files**: `frontend/src/features/auth/{LoginPage,RegisterPage}.tsx`,
  `frontend/src/features/projects/ProjectsPage.tsx`,
  `frontend/src/features/tasks/{TaskListPage,TaskDetailModal}.tsx`,
  `frontend/src/features/actionplan/ActionPlanPage.tsx`, `frontend/src/App.tsx`.

### [MEDIUM] Query error states are largely unhandled — silent blank/stuck screens
- **Current implementation**: A global error handler exists only for
  **mutations** (`MutationCache.onError`, `App.tsx:14-20`); `useQuery` calls
  across nearly every feature never check `isError`/`error`, only
  `isLoading`/`data`. A failed `GET` leaves the page stuck on a loading skeleton
  or renders an empty page once loading flips false with `data` still `undefined`.
- **Recommended solution**: Add a `QueryCache` global `onError` parallel to the
  existing `MutationCache` one; add `isError` branches (reusing the existing
  `EmptyState` component) to the pages currently missing them.
- **Files**: `frontend/src/App.tsx`,
  `frontend/src/features/{team,gamification,reports,dashboard}/*Page.tsx`,
  `frontend/src/routes/ProjectLayout.tsx`.

### [MEDIUM] Zero automated frontend tests
- **Current implementation**: No `vitest`/`@testing-library/react`/`playwright`/
  `cypress` in `package.json`; no `*.test.tsx` files anywhere.
- **Risk**: Zero coverage of the token-refresh logic in `lib/api.ts` — which has
  real concurrency logic (a single in-flight `refreshing` promise dedupe) that is
  exactly the kind of code that silently breaks under refactors — plus the auth
  store, RBAC-gated UI, Kanban drag-drop, and Gantt date/critical-path math.
- **Recommended solution**: Add Vitest + React Testing Library, starting with
  `lib/api.ts`'s refresh-and-retry logic and `auth-store.ts`; add Playwright for
  at least one E2E smoke path (login → create project → create task → drag on
  Kanban).
- **Files**: New `frontend/vitest.config.ts`, `lib/api.test.ts`,
  `stores/auth-store.test.ts`, `package.json`.

### [MEDIUM] LINE Notify token entered in a plain-text (non-masked) input
- **Current implementation**: `NotificationPreferencesPage.tsx:54` — a plain
  `<Input>` with no `type="password"` for a personal access token.
- **Recommended solution**: Mask with a show/hide toggle; backend should never
  echo the stored token back in plaintext on `GET` (return a masked placeholder).
- **Files**: `frontend/src/features/notifications/NotificationPreferencesPage.tsx`;
  backend `GET /notification-preferences` response shaping.

### [LOW] Performance/consistency items (grouped)
- **No memoization/virtualization** on `KanbanBoard`/`TaskListPage`/`GanttPage` —
  fine at current scale, will get janky at hundreds of cards per column;
  `React.memo` + `@tanstack/react-virtual` if that scale is expected.
- **Inconsistent localStorage usage** — theme/sidebar-collapse state bypasses the
  Zustand `persist` pattern used elsewhere; fold into a small store for
  consistency (no security impact, maintainability only).
- **Minimal accessibility coverage** — 5 total `aria-*`/`role=`/`alt=` occurrences
  codebase-wide; Kanban drag-and-drop has no keyboard alternative (though the
  list view's `<Select>` fallback for status changes is a good existing mitigant).
- **No i18n** — single-locale today; flagged per the audit ask, out of scope
  unless multi-locale becomes a roadmap requirement.
- **Hardcoded relative `/api/v1` base URL** — reasonable for the current
  same-origin nginx topology; add `VITE_API_BASE_URL` only if multi-environment
  deployments against different backends becomes a real need.

### Confirmed correct (do not rewrite)
- **Centralized axios client with correct refresh-and-retry**: `lib/api.ts`
  properly deduplicates concurrent 401s behind a single `refreshing` promise,
  retries once, and falls back to a clean logout on refresh failure — genuinely
  well-built, only undermined by *where* tokens are stored (§1).
- **No client-side EXP/points/role tampering surface** — gamification is entirely
  read-only from the frontend; role changes go through a dedicated endpoint, not
  a generic object-update endpoint.
- **Clean feature-sliced architecture** — consistent `api.ts`/`types.ts`/`Page.tsx`
  structure per feature, no cross-feature coupling, TanStack Query correctly used
  as the source of truth for server state (avoiding a common Zustand-mirroring
  staleness bug).
- **Consistent loading/empty states** via shared `Skeleton`/`EmptyState`
  components reused across features.
- **AI feature is entirely backend-proxied** — no LLM provider keys ever exposed
  client-side, confirmed via grep.
- **Realtime updates use a deliberately simple invalidate-on-message pattern**
  rather than trying to patch the query cache from partial socket payloads — a
  defensible, low-bug-surface choice, documented in-code.

---

## 14. Testing
*(covers category: 21 Testing)*

### [MEDIUM] No test coverage measurement/threshold in CI; frontend has zero test execution in CI
- **Current implementation**: `.github/workflows/ci.yml:22` runs `go test ./...`
  with no `-cover`/`-coverprofile`; the frontend job runs only `npm run lint` and
  `npm run build` — no test step at all (consistent with the zero-tests finding
  in §13).
- **Recommended solution**: Add `go test ./... -coverprofile=coverage.out` with a
  minimum-percentage gate (or Codecov); once frontend tests exist (§13), wire them
  into CI with coverage reporting.
- **Files**: `.github/workflows/ci.yml`, `frontend/package.json`.

### Confirmed correct (do not rewrite)
- **Backend test coverage is genuinely solid** — 13 unit-test files across every
  module using well-built hand-rolled fakes (no framework-mocking magic),
  covering branchy business logic: streak resets, badge thresholds, mission
  one-time rewards, membership checks, error propagation, dependency-cycle
  detection, critical-path edge cases.
- **A real E2E suite exists** (`backend/e2e/e2e_test.go`,
  `scenarios_test.go`) running against actual Postgres+Redis service containers
  in CI, not just mocked units — covers first-user-is-superadmin bootstrap
  ordering, cross-module gamification wiring, permission boundaries, and
  comment→notification wiring. This is meaningfully better than a typical
  unit-only suite and should be extended, not replaced.

---

## 15. Docker, CI/CD, Production Configuration & Environment Management
*(covers categories: 22 Docker, 23 CI/CD, 27 Production configuration, 28 Environment management)*

### [HIGH] No dedicated reverse-proxy / TLS-termination layer despite the architecture spec requiring one
- **Current implementation**: `docker-compose.yml` defines only `postgres`,
  `redis`, `minio`, `api`, `web` — no `nginx` service, despite
  `system-architecture-design.md` explicitly describing one that owns TLS
  termination. What exists instead is `frontend/nginx.conf`, proxying `/api`/`/ws`
  from inside the `web` container to `api:8080` — but only on plain port 80, no
  `ssl_certificate`/443 listener anywhere in the repo.
- **Risk**: Deploying this compose file as-is to a public server serves the
  entire app over plain HTTP — credentials, JWTs, session data transit in
  cleartext.
- **Recommended solution**: Either add a dedicated TLS-terminating proxy
  (nginx/Caddy/Traefik with Let's Encrypt/ACME) in front of `web`+`api`, or, if
  the frontend-embedded nginx is the intended final design, add TLS termination
  there and update the architecture doc to match reality.
- **Files**: `docker-compose.yml`, `frontend/nginx.conf` or a new `nginx/`
  service, new `docker-compose.prod.yml`, `docs/superpowers/specs/system-architecture-design.md`.

### [HIGH] CI has no image build/push, no deploy step, no dependency/image vulnerability scanning
- **Current implementation**: `.github/workflows/ci.yml` has exactly 3 jobs
  (backend build/vet/test, e2e, frontend lint/build) — no `docker build`, no
  registry push, no deploy job, no `govulncheck`/`npm audit`/Trivy/CodeQL step
  anywhere.
- **Risk**: Deployment is presumably manual and untracked; vulnerable
  dependencies (Go modules, npm packages, or base image CVEs) can ship silently.
- **Recommended solution**: Add a `docker` job building and pushing tagged images
  to a registry on `main`; add `govulncheck ./...` to the backend job and
  `npm audit --audit-level=high` to the frontend job; add a Trivy/Grype scan
  against built images; add a CD job (or documented runbook) for rolling deploy.
- **Files**: `.github/workflows/ci.yml` (or new `cd.yml`).

### [HIGH] No environment separation — a single flat compose/.env for dev, staging, and prod
- **Current implementation**: Only one `docker-compose.yml` exists — no
  `docker-compose.prod.yml`/`.staging.yml`/`.override.yml`. Postgres/Redis/MinIO
  are all bound directly to host ports (fine for local dev, a real exposure risk
  on a naively deployed VPS, bypassing all app-layer auth). No service anywhere
  has a `restart` policy.
- **Recommended solution**: Base `docker-compose.yml` (dev-safe defaults) plus a
  `docker-compose.prod.yml` override that removes host port publishing for
  datastores (internal Docker network only), adds `restart: unless-stopped`, sets
  resource limits, and wires in the TLS proxy above. Add `.env.staging.example`/
  `.env.production.example` templates and a runbook for which compose command to
  run per environment.
- **Files**: New `docker-compose.prod.yml`, updated `Makefile` targets, new env
  templates, deployment docs.

### [MEDIUM] Cluster of smaller Docker/ops hygiene gaps
- Containers run as root in both `Dockerfile`s (no `USER` directive); no
  `.dockerignore` anywhere (risk of `.env`/`.git` landing in build context via
  `COPY . .`). Fix: add a non-root user + `.dockerignore`.
- Secrets are plain environment variables with no secrets manager — reasonable
  for the current single-VPS scale, flagged only as a future hardening step tied
  to the environment-separation work above, not urgent today.

### Confirmed correct (do not rewrite)
- **Multi-stage Dockerfiles** on both backend and frontend, small final images,
  `CGO_ENABLED=0` static Go build.
- **Compose healthchecks for stateful deps** (`postgres`, `redis`) with
  `condition: service_healthy` gating the API — a decent guardrail, just needs
  extending to the `api` service itself once `/readyz` exists (§10).
- **Required secrets enforced at compose level** via `${VAR:?}` syntax — compose
  refuses to start with required secrets unset.
- **A real E2E job in CI** running against genuine Postgres+Redis service
  containers, not mocks — see §14.

---

## Master severity table

| # | Finding | Section | Severity |
|---|---|---|---|
| 1 | No backup/recovery for Postgres/MinIO | §11 | **CRITICAL** |
| 2 | No panic-recovery middleware | §4 | **CRITICAL** |
| 3 | No rate limiting anywhere (login/register/refresh/global) | §4 | **CRITICAL** |
| 4 | JWT tokens in `localStorage` | §1 | **CRITICAL** |
| 5 | No versioned migrations (AutoMigrate at boot) | §3 | **CRITICAL** |
| 6 | Kanban/project list silently truncates at 20 items | §5 | **CRITICAL** |
| 7 | Access token in WebSocket URL query string | §1 | HIGH |
| 8 | No foreign key constraints | §3 | HIGH |
| 9 | No soft deletes | §3 | HIGH |
| 10 | No DB connection pool limits | §3 | HIGH |
| 11 | No transactions on multi-step writes | §3 | HIGH |
| 12 | EXP read-modify-write race condition | §7 | HIGH |
| 13 | EXP farming via status toggling | §7 | HIGH |
| 14 | No audit logging | §12 | HIGH |
| 15 | No security headers (helmet/CSP/HSTS) | §4 | HIGH |
| 16 | Overly permissive CORS | §4 | HIGH |
| 17 | No graceful shutdown | §4 | HIGH |
| 18 | No Multi-Org/Department/Team hierarchy | §2 | HIGH |
| 19 | Report/Gantt N+1 + full in-memory aggregation | §5 | HIGH |
| 20 | Notification delivery synchronous, no queue | §6 | HIGH |
| 21 | No notification retry/dead-letter/delivery tracking | §6 | HIGH |
| 22 | No AI rate limiting/cost control | §8 | HIGH |
| 23 | No global React error boundary | §13 | HIGH |
| 24 | Frontend `ProtectedRoute` has no role checks | §2 | HIGH |
| 25 | No TLS-terminating reverse proxy | §15 | HIGH |
| 26 | CI: no image build/deploy/security scanning | §15 | HIGH |
| 27 | No environment separation (dev/staging/prod) | §15 | HIGH |
| 28 | `RequireRole` middleware unused, no route RBAC defense-in-depth | §2 | MEDIUM |
| 29 | JWT parsing doesn't pin signing algorithm | §1 | MEDIUM |
| 30 | No Fiber Read/Write/Idle timeouts | §4 | MEDIUM |
| 31 | No structured logging / request ID | §4 / §10 | MEDIUM |
| 32 | Password-reset flow half-defined, not implemented | §1 | MEDIUM |
| 33 | Missing indexes (status, due_date, notification read) | §3 | MEDIUM |
| 34 | No optimistic concurrency on Task update | §3 / §5 | MEDIUM |
| 35 | TOCTOU race in dependency-cycle check | §5 | MEDIUM |
| 36 | Team directory N+1 workload/presence queries | §5 | MEDIUM |
| 37 | No file-type allow-list / malware scan on attachments | §5 | MEDIUM |
| 38 | No recurring-task support | §5 | MEDIUM |
| 39 | LINE integration: no backoff, plaintext token, possible deprecation | §6 | MEDIUM |
| 40 | Cron digests: no distributed lock (duplicate on scale-out) | §6 | MEDIUM |
| 41 | Digest cron queries unbounded, no batching | §6 | MEDIUM |
| 42 | AI calls synchronous, 60s timeout, no circuit breaker | §8 | MEDIUM |
| 43 | No form validation library on frontend | §13 | MEDIUM |
| 44 | Frontend query errors mostly unhandled (blank/stuck screens) | §13 | MEDIUM |
| 45 | Zero automated frontend tests | §13 / §14 | MEDIUM |
| 46 | LINE token plaintext input on frontend | §13 | MEDIUM |
| 47 | `/health` is a liveness stub, not real readiness | §10 | MEDIUM |
| 48 | No metrics/Prometheus | §10 | MEDIUM |
| 49 | No error tracking/APM | §10 | MEDIUM |
| 50 | No test coverage measurement/threshold in CI | §14 | MEDIUM |
| 51 | Docker hygiene: root containers, no `.dockerignore` | §15 | MEDIUM |
| 52 | bcrypt cost at library default (10) | §1 | LOW |
| 53 | GORM logs full SQL with values at Warn | §4 | LOW |
| 54 | MinIO credentials silently default to empty string | §4 | LOW |
| 55 | `GetLeaderboard` O(n²) sort, no pagination/caching | §6 | LOW |
| 56 | Notification/gamification errors silently discarded, no metric | §6 | LOW |
| 57 | No intra-column Kanban ordering (feature gap) | §5 | LOW |
| 58 | No workflow state-machine on task status | §5 | LOW |
| 59 | Unsanitized filename in object storage key | §5 | LOW |
| 60 | No sort option; non-indexable `ILIKE` search | §5 | LOW |
| 61 | Prompt injection surface (bounded impact today) | §8 | LOW |
| 62 | Frontend: no memoization/virtualization at scale | §13 | LOW |
| 63 | Frontend: inconsistent localStorage usage patterns | §13 | LOW |
| 64 | Frontend: minimal accessibility coverage | §13 | LOW |
| 65 | Frontend: no i18n | §13 | LOW |
| 66 | Secrets as plain env vars, no secrets manager | §15 | LOW |

---

## Prioritized implementation roadmap

### P0 — Security / Data Loss / Production Blocking (do first, in roughly this order)
1. **Backups** for Postgres + MinIO with a tested restore drill (#1) — nothing
   else matters if data can vanish.
2. **Panic recovery middleware** (#2) and **rate limiting** (#3) — two-line-ish
   fixes that close a process-crash DoS and an unlimited-brute-force hole.
3. **Kanban/list pagination fix** (#6) — a present-day correctness bug, not a
   scale concern; fix before anything else user-facing.
4. **Versioned migrations** replacing boot-time `AutoMigrate` (#5), and layer on
   **FK constraints** (#8) and **soft deletes** (#9) in the same migration effort
   — these three are most efficiently done together since soft-deletes and FKs
   both need real migration files anyway.
5. **DB connection pool limits** (#10) and **transactions around multi-step
   writes** (#11).
6. **Token storage hardening**: move JWTs out of `localStorage` to `httpOnly`
   cookies (#4) and get the WS handshake token out of the URL (#7) — do these
   together since both touch the same auth/session surface.
7. **Gamification integrity**: fix the EXP race condition (#12) and the EXP
   farming exploit (#13) together, since both touch `OnTaskCompleted`.
8. **Security headers + CORS allowlist** (#15, #16), **graceful shutdown** (#17).
9. **Audit logging** (#14) — do alongside soft-deletes (#9) so deletes get an
   actor record from day one.
10. **AI rate limiting/cost control** (#22) before any real user traffic, given
    direct financial exposure.
11. **TLS-terminating proxy** (#25) and **environment separation** with
    datastore ports off the public network (#27) — required before any
    production deployment regardless of code changes.
12. **CI security scanning + deploy pipeline** (#26).

### P1 — Core Enterprise Features
- **Multi-Organization / Department / Team schema** (#18) — the largest single
  effort in this list; start early since it touches nearly every module's
  queries and authorization checks, and everything built on top of the current
  flat schema will need re-validation once it lands.
- **Notification Queue**: async delivery (#20) + retry/dead-letter/delivery
  tracking (#21) — this is the literal "Notification Queue" requirement.
- Report/Gantt aggregate-SQL rewrite + N+1 fixes (#19).
- Frontend error boundary (#23) and route-level RBAC (#24).
- Wire `RequireRole` as defense-in-depth (#28).
- Password-reset flow (#32).
- Optimistic locking on tasks (#34).
- File-upload type allow-list (#37).
- Metrics/Prometheus (#48) and error tracking/APM (#49).
- Test coverage in CI, backend and frontend (#50), plus standing up a frontend
  test suite (#45).
- Frontend form validation (#43).

### P2 — Reliability / Performance
- Missing DB indexes (#33), team-directory N+1 fixes (#36).
- Cron distributed locking (#40) and digest batching (#41) — matters once
  horizontally scaled.
- LINE integration hardening: backoff, token encryption, deprecation check (#39).
- AI circuit breaker/timeout tuning (#42).
- Real `/readyz` liveness/readiness split (#47).
- Structured logging + request-ID correlation (#31).
- Frontend query-error handling (#44), LINE token masking (#46).
- Leaderboard pagination/caching (#55), sort/search indexing (#60).
- Docker hygiene: non-root containers, `.dockerignore` (#51).
- Fiber timeouts (#30), JWT algorithm pinning (#29).
- TOCTOU dependency-cycle fix (#35).

### P3 — UX / Advanced Features
- Recurring tasks (#38).
- Kanban intra-column ordering (#57), workflow state machine (#58) — only if
  product direction wants enforced Kanban gates.
- Prompt-injection prompt hardening (#61).
- bcrypt cost bump (#52), GORM log redaction (#53), MinIO secret consistency (#54).
- Filename sanitization on uploads (#59).
- Secrets manager migration (#66) — revisit once environment separation (P0 #11)
  is in place.
- Frontend polish: localStorage consistency (#63), virtualization/memoization at
  scale (#62), accessibility (#64), i18n (#65).
- Silent-error metric emission for notification/gamification call sites (#56).

---

## Notes on scope preservation

Per standing project rules, this roadmap **does not** propose microservices,
a framework rewrite, or discarding the existing Clean Architecture / Repository
Pattern / Service Layer structure — all of it is sound and should be built on top
of, not replaced. The single largest structural change recommended (§2, Multi-Org/
Department/Team) is additive to the existing modular-monolith shape: new tables
and scoping columns, not a new architecture.
