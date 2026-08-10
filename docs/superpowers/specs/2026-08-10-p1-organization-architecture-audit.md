# P1 Phase 1 — Architecture Audit: Single-Company → Multi-Organization

Date: 2026-08-10
Status: Audit complete — awaiting approval before implementation
Scope: research only, no code or schema changed.

This audit inspects the current implementation as it exists **today**, after
P0.1 (backup/recovery) and P0.2 (panic recovery + rate limiting) — P0.3
onward (including P0.4, which introduces versioned migrations) has **not**
been implemented yet. That gap turns out to matter for this phase — see
§9 and the flagged blocker below.

## Read before anything else: two premises in the P1 brief don't match current state

1. **"Audit Logs" is listed among existing features to preserve — it does not
   exist.** A repo-wide search for `audit` across the backend returns zero
   matches. There is no audit log table, service, or middleware anywhere.
   This was flagged as a gap (§12, HIGH) in the original production-readiness
   audit and hasn't been built. P1's "extend the existing audit system"
   (Phase 15) is actually "build the audit system," full stop — worth
   knowing before scoping that phase's effort.
2. **Phase 18 says "use the versioned migration system implemented in P0"
   — it isn't implemented yet.** The original P0 roadmap's P0.4 ("Versioned
   migrations + FK constraints + soft deletes") is what introduces that
   tooling. We've only completed P0.1 and P0.2 so far; schema is still
   managed by `bootstrap.Migrate()` calling GORM `AutoMigrate()` at every
   process boot (`backend/internal/bootstrap/bootstrap.go:104-115`), same as
   it was at the start of the audit. **This is a real dependency, not a
   nitpick**: P1.1 is titled "Organization + Migration," and it cannot
   honestly produce a "versioned migration" for the organization schema
   without a migration tool existing first. I'm flagging this now so you can
   decide — see the recommendation at the end of §9 — rather than silently
   either (a) writing P1.1's migration as another `AutoMigrate` change
   (violating Phase 18's explicit instruction) or (b) quietly reordering
   your roadmap without telling you.

Everything else below reflects the actual current code.

---

## 1. Current schema

Schema is defined entirely via GORM struct tags across module `model.go`
files, applied by `AutoMigrate` — no SQL migration files exist. Tables as
they exist today (grouped by module):

**auth**
- `users(id uuid pk, name, email uniqueIndex, password_hash, role varchar(20) default 'employee', avatar_url, department_id uuid index nullable, created_at, updated_at)`
  — `role` is a single flat string enum: `super_admin | admin | manager | leader | employee`. No org scoping of any kind.

**project**
- `projects(id uuid pk, name, description, owner_id uuid index, status varchar(20), due_date, color, icon, created_at, updated_at)`
- `members(project_id uuid pk-part, user_id uuid pk-part, role varchar(20) default 'member', created_at)` — composite PK `(project_id, user_id)`. `role` is `owner | member` only (not the `owner|member|viewer` implied by Phase 11's target, and not the 3-tier `PROJECT_MANAGER|MEMBER|VIEWER` Phase 2 wants).

**task**
- `tasks(id uuid pk, project_id uuid index, parent_task_id uuid index nullable, title, description, priority varchar(20), status varchar(20), start_date, due_date, estimate_hours, assignee_id uuid index nullable, reporter_id uuid not-null, milestone_id uuid index nullable, completed_at, created_at, updated_at)`
- `checklist_items(id pk, task_id index, text, done, position, created_at)`
- `tags(task_id pk-part, tag pk-part)`
- `dependencies(task_id pk-part, depends_on_id pk-part, created_at)`
- `comments(id pk, task_id index, author_id, body, created_at)`
- `attachments(id pk, task_id index, uploader_id, file_name, content_type, size_bytes, object_key, created_at)`
- `watchers(task_id pk-part, user_id pk-part, created_at)`

**actionplan**
- `goals(id pk, project_id index, name, description, status, due_date, created_at, updated_at)`
- `milestones(id pk, goal_id index, name, description, status, due_date, created_at, updated_at)`

**team**
- `departments(id pk, name uniqueIndex, created_at)` — flat, global, unscoped. No `Team` entity exists at all; the `team` Go package is a directory/composition layer (auth + task + presence), not an owner of a `teams` table.

**notification**
- `notifications(id pk, user_id index, type varchar(30), title, body, link, read, created_at)`
- `preferences(user_id pk, email_enabled, line_enabled, line_notify_token)`

**gamification**
- `characters(user_id pk, exp, level, total_completed, current_streak, longest_streak, last_completion_date, created_at, updated_at)`
- `badges(user_id pk-part, code pk-part, awarded_at)`
- `mission_progresses(user_id pk-part, type pk-part, period_key pk-part, count, rewarded, updated_at)`

**Nothing exists yet for**: `organizations`, `departments.organization_id`, `teams` (table), any `team_members`, `organization_members`, `invitations`, `audit_logs`, or a permission table of any kind.

**No foreign key constraints exist anywhere** — every relationship above (`project_id`, `user_id`, `assignee_id`, `department_id`, etc.) is a bare indexed UUID column with referential integrity enforced only in application code, never at the database level. This was already flagged in the original audit (§3, HIGH) and matters directly for this migration: adding `organization_id` FKs is the first time this schema will have *any* real FK constraints, so get the constraint-adding migration right the first time rather than layering more unenforced columns on top of already-unenforced ones.

---

## 2. Current authorization flow

```
Request
 → Authorization: Bearer <JWT>
 → middleware.RequireAuth (backend/internal/middleware/auth.go:13-29)
     parses JWT, sets c.Locals("userID"), c.Locals("role")
 → handler
     httpctx.ActorID(c) / httpctx.ActorRole(c)  (pkg/httpctx/httpctx.go:15-31)
 → service layer (per-module, ad hoc, see below)
 → repository (plain GORM queries, no scoping)
```

Two authorization primitives exist, used inconsistently:

1. **`auth.IsOrgAdmin(role)`** (`auth/role.go:5-7`) — returns true for
   `admin` or `super_admin`. Every service that needs an "is this user
   privileged" check calls this directly. Critically: **this is a truly
   global bypass today** — an `admin` can manage *every* project, *every*
   task, *every* user in the entire system, because there is no
   organization boundary to scope it to. This is the single most important
   thing that changes in a multi-org model: today's `admin` semantically
   *is* what the target state calls `ORGANIZATION_ADMIN`, but it currently
   has no organization to be scoped to.

2. **Per-resource membership checks**, implemented independently in each
   module's service, not shared:
   - `project.Service.canManage` / `CheckMembership`
     (`project/service.go:249-275`) — org admin bypass, else looks up
     `members` row for `(project_id, actor_id)`.
   - `task.Service.requireMembership` / `requireCanEdit`
     (`task/service.go:1057-1086`) — delegates to `project.Service.CheckMembership`.
   - `ai.Service.requireMembership` (`ai/service.go:45`) — same pattern,
     independently implemented.
   - `actionplan`, `report` — go through `project.Service` (constructor
     injection), don't re-implement the check.

3. **`middleware.RequireRole(...roles)`** (`middleware/auth.go:33-46`) is
   fully implemented but **never called anywhere** — confirmed by grep,
   zero call sites outside its own definition. All authorization today
   happens in the service layer via #1/#2 above, not in route-level
   middleware. (Flagged in the original audit, §2, MEDIUM — relevant here
   because Phase 6's `requirePermission(...)` /
   `requireOrganizationAccess()` will be the first time this codebase
   actually uses middleware-layer authorization for anything beyond "is
   there a valid token.")

There is **no concept of "current organization"** anywhere in the request
lifecycle — nothing to derive it from, nothing to check it against, no
column to filter by. Every list/query in every module operates over the
entire, single, global dataset.

---

## 3. Tables requiring `organization_id`

**Stored directly** (recommended — these are the tables every authorization
check and every list/dashboard query needs to filter by first, and forcing
a join through a parent chain for every single query would be both a
performance tax and an easy place to forget the join and leak data):

| Table | Why direct, not derived |
|---|---|
| `organizations` (new) | is the root |
| `users` | a user's org membership is looked up on every authenticated request (JWT → org context, Phase 3's diagram) — this needs to be a direct, indexed column, not a join |
| `departments` | org-owned directly per Phase 2's hierarchy |
| `teams` (new) | org-owned directly (also `department_id`, see §5) |
| `projects` | org-owned directly — every project list/dashboard query filters by org first |

**Derived through parent, not stored** (per Phase 4's own guidance — "avoid
redundant organization_id where it creates consistency problems"):

| Table | Derived via |
|---|---|
| `tasks`, `goals`, `milestones` | → `project.organization_id` |
| `checklist_items`, `tags`, `dependencies`, `comments`, `attachments`, `watchers` | → `task.project.organization_id` |
| `members` (project membership) | → `project.organization_id` (and the member's `user.organization_id` must match — see §8) |
| `notifications` | → `user.organization_id` (a notification's owner already implies its org) |
| `characters`, `badges`, `mission_progresses` (gamification) | → `user.organization_id` |

Phase 4 explicitly allows storing `organization_id` directly on these for
performance, with the caveat that the backend must then guarantee
`task.organization_id == task.project.organization_id` always. My
recommendation: **don't**, at least not in P1.1–P1.7. This schema has zero
FK enforcement today (see §1) and no versioned migration tooling yet either
— adding a second, independently-writable copy of `organization_id` on
every child table multiplies the surface for exactly the
inconsistency bug Phase 4 warns about, for a performance win that doesn't
matter yet at this data volume. `tasks.project_id` is already indexed, so
`WHERE tasks.project_id IN (SELECT id FROM projects WHERE organization_id = ?)`
(or, more likely in practice, resolving the caller's org's project IDs once
per request and filtering by `project_id IN (...)`) is a well-indexed query.
Revisit denormalizing `organization_id` onto `tasks` later, as a P2-scope
performance change, only if profiling shows it's actually needed — not
preemptively.

---

## 4. Tables requiring `department_id`

- `users.department_id` — **already exists** (`auth/model.go:30`, nullable,
  indexed). Needs an added FK once `departments` has real constraints and,
  post-migration, should probably also carry an implicit
  `department.organization_id == user.organization_id` consistency
  requirement, enforced in the service layer (see §8).
- `teams.department_id` (new table, new column) — per Phase 2's hierarchy,
  every team belongs to exactly one department.

No other existing table needs `department_id` directly — everything else
that cares about department (e.g. "this project's department") would go
through `team_id` → `department_id` or through the user, not be duplicated
onto tasks/projects.

---

## 5. Tables requiring `team_id`

There is **no `teams` table today** — Phase 2's Team entity is entirely
new. Once it exists:

- No existing table needs a *direct* `team_id` column for P1.1–P1.6's
  scope. Team's usefulness (workload views, team dashboards — Phases 10/12)
  comes from a **new join table**, `team_members(team_id, user_id, ...)`,
  analogous to `project.Member` — Phase 2's spec doesn't list this table
  explicitly but Phase 10 ("Add/remove members" for teams) requires it to
  exist. Flagging this as a table Phase 2's own model list is missing, so
  it doesn't get discovered as a surprise mid-implementation.
- Whether `projects` ever gets a `team_id` (i.e., "this project belongs to
  this team") isn't specified anywhere in the brief's hierarchy diagram
  (Organization → Projects is drawn as direct, not through Team) — recommend
  leaving projects org-scoped-only (not team-scoped) unless you want that
  relationship; note it here so it's a deliberate decision, not an
  oversight.

---

## 6. Tables requiring project membership

`project.Member` already exists and already does what Phase 2's "Project
Membership" model asks for, structurally: `(project_id, user_id, role,
created_at≈joined_at)`. It needs to be **migrated, not created**:

- Current `role` values: `owner | member` (2 values).
- Target `role` values (Phase 11): `PROJECT_MANAGER | MEMBER | VIEWER` (3
  values, and no bare "owner" concept — project management authority
  becomes a role value, not a separate flag).
- Data migration needed: existing `role = 'owner'` rows → map to
  `PROJECT_MANAGER`; existing `role = 'member'` rows → map to `MEMBER`;
  `VIEWER` is new and nothing maps to it (fine — it's opt-in going
  forward).
- `project.Project.OwnerID` (a separate column, not part of `Member`) also
  needs a decision: keep it as "who created this project" metadata, or
  fold it entirely into the `PROJECT_MANAGER` membership role and drop the
  column. Recommend keeping `OwnerID` as immutable creation metadata (cheap,
  already there, useful for "created by" display) while `Member.Role`
  becomes the authoritative access-control signal — don't conflate the two
  meanings.

---

## 7. Existing permission model

There isn't one, in the sense Phase 5/6 mean. Today: **5 flat, global,
string-valued roles** (`super_admin, admin, manager, leader, employee`),
checked via `auth.IsOrgAdmin()` (a hardcoded 2-role bypass list) or ad hoc
per-service membership lookups (§2). No permission strings
(`task.update`), no permission table, no role→permission mapping, no
`requirePermission(...)` primitive anywhere in the codebase.

Rough mapping from current → target roles (for the migration's data
transform, §9):

| Current | Target | Notes |
|---|---|---|
| `super_admin` | **splits in two** | Today's "first user ever registered" (`auth/service.go:75-79`, `bootstrapRole`) becomes `super_admin` globally — that specific bootstrap behavior must change (see §9); going forward `SUPER_ADMIN` should be a deliberately-provisioned, cross-org, rare role, not an automatic grant |
| `admin` | `ORGANIZATION_ADMIN` | closest existing match — today's admin already behaves like an all-projects admin, just without an org boundary |
| `manager` | `PROJECT_MANAGER` | reasonable direct mapping |
| `leader` | `TEAM_LEADER` | reasonable direct mapping, though today's "leader" has no team-scoped meaning yet (no Team entity) — it's currently just another flat global role |
| `employee` | `EMPLOYEE` | direct mapping |
| *(none)* | `VIEWER` | new, nothing maps to it |

None of today's 5 roles have any org/project/team *scope* attached — a
`manager` is a manager of everything, not of specific projects (project-level
authority comes entirely from `project.Member.Role`, a completely separate
mechanism). The target model's role scoping (Phase 7 — "manage **assigned**
projects", "manage **assigned** teams") is new behavior, not currently
present in any form.

---

## 8. Potential data isolation risks

These are the concrete places a cross-organization leak would happen if
org-scoping is bolted on incompletely — i.e., the actual checklist for
Phase 3/Phase 21's testing:

1. **`auth.IsOrgAdmin` bypass is currently global and must become
   org-scoped.** The single highest-risk item: if `ORGANIZATION_ADMIN`
   authorization is implemented as "just check the role string, same as
   today" without also checking *which* organization the actor's admin
   privilege applies to, every org admin becomes a de facto super admin
   over every organization. This is the most likely place a first
   implementation attempt gets it wrong, because it's the path of least
   resistance (the existing `IsOrgAdmin` check is a one-line role compare;
   the correct version needs the actor's org membership row too).
2. **Every unscoped `List*`/`Find*` query becomes a cross-org leak the
   moment `organization_id` exists but isn't filtered.** Concretely, as of
   today: `project.Repository.ListForMember` / equivalents, `task.Repository.List`
   and `ListAllForProject`, `team.Repository.ListDepartments` (currently
   returns literally every department in the database, no scope at all —
   `team/repository.go`), `auth.Repository`'s `ListUsers` (returns every
   user globally — used by `team.Service.GetDirectory` and by the
   notification digest cron). Each of these needs an explicit org filter
   added, not inherited automatically — GORM does not scope queries for
   you.
3. **Gamification leaderboard is global today** (`gamification.Repository.ListCharacters`,
   no `LIMIT`, no scope) — direct information disclosure across orgs once
   multiple orgs exist (org A's employees would see org B's EXP/leaderboard
   standings).
4. **Notification digest crons iterate every user in the system**
   (`notification.Service`'s `forEachUser` pattern, calling
   `authSvc.ListUsers` unscoped) — needs org-aware batching once
   `organization_id` exists, or digests silently ignore org boundaries.
5. **AI Assistant endpoints** (`ai.Service`) check *project* membership
   (`requireMembership`) but have no organization concept at all today —
   once orgs exist, a project's organization must be verified consistent
   with the actor's, not just "is this person a member of this specific
   project row" (which becomes necessary-but-not-sufficient once project
   IDs are no longer implicitly scoped to one company).
6. **Realtime WebSocket channels** (`project:<id>`, `user:<id>` topics in
   `internal/realtime`) are keyed by UUID, not by org — not a leak by
   itself (UUIDs aren't guessable/enumerable), but worth a deliberate note:
   the WS auth handshake (`realtime/handler.go`) validates the JWT and,
   implicitly via existing project-membership checks before subscribing, so
   this should be safe *if* the subscribe-time membership check is
   correctly org-aware — another place the same "membership check must also
   check org" pattern from #1 applies.
7. **Cross-table consistency drift** (Phase 4's own concern): once
   `organization_id` exists anywhere derived-vs-stored, a bug that lets
   `task.project_id` point at a project in a different org than intended
   (e.g. via an unvalidated `project_id` in a request body — mass
   assignment / parameter tampering, explicitly called out in Phase 21)
   would silently move a task across the org boundary. Every write path
   that accepts a foreign-key ID from the client (task's `project_id` on
   create, a comment's implicit `task_id`, etc.) needs the same treatment
   auth already gets: **never trust a client-supplied ID to imply
   authorization — re-derive and re-verify server-side on every write**,
   which Phase 3's diagram states as a principle but which needs to be
   applied literally to every existing create/update handler, not just new
   organization endpoints.
8. **First-user-becomes-super-admin bootstrap** (`auth/service.go:75-79`)
   is itself a data-isolation-adjacent risk in a multi-org world if left
   unchanged: it currently means "the first person to ever call
   `/auth/register` on this deployment" — in a multi-org SaaS-like model,
   that behavior needs to change to "the first person to register **within
   a given organization**, if self-serve org creation is allowed" (or be
   replaced entirely by the invitation flow, Phase 8) — otherwise the
   *second* organization's first user would also incorrectly get global
   `super_admin`.

---

## 9. Migration strategy

**Blocking dependency, restated plainly**: this section assumes P0.4
(versioned migrations) lands first, or is folded into P1.1. See the
recommendation at the end.

Once a migration tool exists, the shape of the migration itself (standard,
low-risk pattern for retrofitting a tenant column onto live data):

1. **Additive schema changes only, nothing destructive, in the first
   migration**: create `organizations` table; add `organization_id` as a
   **nullable** UUID column (no FK yet, no NOT NULL yet) to `users`,
   `departments`, `projects`.
2. **Data backfill, as a second migration/step**: create exactly one
   "default organization" row (e.g. named after whatever this deployment's
   company is, or a placeholder like "Default Organization" — needs a real
   name, not a config-file guess, so this step should be a reviewed,
   deployment-specific value, not baked into the migration file verbatim).
   Assign every existing `users` row, every existing `departments` row,
   every existing `projects` row to that organization's ID.
3. **Verify consistency before tightening constraints**: a query pass
   confirming every `users.organization_id`, `departments.organization_id`,
   `projects.organization_id` is now non-null, and (once `members` is
   considered) every project member's user shares the project's
   organization — this repo has zero existing data that could violate that
   (single company, single implicit org), so this step should be a no-op
   assertion, not an actual data-fixing step, for the very first run. It
   still needs to exist as a migration-time check (not just an assumption)
   because it's the thing that would catch a mistake in step 2.
4. **Only then**, in a follow-up migration: make `organization_id` `NOT
   NULL`, add the real FK constraints (`REFERENCES organizations(id)`), add
   the composite unique constraints the brief specifies
   (`organization_id + user_id` for org membership,
   `organization_id + department.name`, `department_id + team.name`,
   `project_id + user_id` for project membership — already effectively
   present as `members`' composite PK).
5. **`users.email` uniqueness needs to change** from global-unique
   (`auth/model.go:22`, `gorm:"uniqueIndex"`) to
   `unique(organization_id, email)` if the same person's email should be
   able to exist in two different organizations as separate accounts — this
   is a real product decision, not just a schema detail, worth confirming
   explicitly before writing the migration (the brief's invitation flow in
   Phase 8 implies email-based invites, which works either way, but the
   uniqueness *scope* changes what "this email is already registered" means
   at `/auth/register`).
6. **Rollout order**: schema migration → backend authorization code deployed
   already org-aware (reading the now-populated `organization_id`) → this
   must ship together, not schema-first-then-code-later, since the moment
   `organization_id` is nullable-but-unenforced there's a window where old
   code paths (unscoped queries) and new org-aware code paths could
   disagree about what's visible. Practically: this argues for the schema
   migration and the first org-scoped read-path change (§8's item 2, the
   unscoped `List*` queries) landing in the **same** deploy, not spread
   across two.
7. **Before running against real data**: take a fresh backup using the
   P0.1 backup system (already built, already verified end-to-end) and
   do a **real restore drill** first — not just trust the automated
   in-pipeline verification — since this is the first migration this
   project will have ever run outside `AutoMigrate`, and it's changing
   uniqueness constraints on `users.email`, which is exactly the kind of
   change that's expensive to get wrong on live data.

**Recommendation on the blocking dependency**: fold the versioned-migration
tooling setup (pick a tool — `golang-migrate` is the most common Go choice
and needs no ORM integration beyond running against the same `DATABASE_URL`
— wire it into `Makefile`/CI, snapshot today's `AutoMigrate`-produced schema
as migration `0001`) into the **start** of P1.1, before the organization
migration itself (which becomes `0002`+). It's a natural fit ("Organization
+ Migration" is literally P1.1's name) and avoids either violating Phase
18's instruction or silently reordering your roadmap without asking. If
you'd rather do the migration-tooling setup as its own separate, reviewable
step before P1.1 starts, that's also reasonable — flagging so you can pick,
not deciding it myself.

---

## 10. Rollback strategy

- **Every forward migration gets a paired down-migration** (standard
  practice for whichever tool is adopted) — for the additive steps (new
  table, new nullable columns) the down-migration is a clean `DROP
  TABLE`/`DROP COLUMN`, genuinely safe. For the constraint-tightening
  migration (NOT NULL, FKs, the `users.email` uniqueness scope change),
  the down-migration needs to reverse the constraint *without* losing the
  now-populated `organization_id` data (i.e., don't drop the column on
  rollback, only relax the constraints) — a partial rollback, which is the
  normal and correct shape for this kind of change.
- **Code rollback must be compatible with the schema state it's rolling
  back to.** Because of the "ship schema + org-aware reads together"
  ordering in §9 item 6, rolling back the *code* to a pre-org-aware
  version while the *schema* still has the new columns is safe (extra
  columns the old code ignores are harmless); rolling back the *schema*
  while org-aware code is still deployed is not (the code would start
  reading a column/table that no longer exists) — so schema rollback and
  code rollback must happen together, in that order: code first, schema
  second, never schema-only.
- **Full backup immediately before the migration runs**, using the
  already-built and already-verified P0.1 backup/restore tooling
  (`make backup`, or the scheduled run if it's already landed in
  production by the time this ships) — this is the actual safety net if
  something goes wrong in a way the down-migrations don't cleanly cover
  (e.g. a partially-applied backfill), and per P0.1's own standard, that
  backup's restorability should be spot-checked, not assumed.
- **A kill-switch is not proposed here as a schema/code feature** (e.g. a
  runtime flag to "disable org scoping") — deliberately: a flag that can
  disable authorization scoping is itself a standing security risk (an
  attacker or a misconfiguration flipping it would silently remove tenant
  isolation). The rollback path for a bad org-scoping deploy should be
  "roll back the deploy," not "flip a switch that turns off the isolation
  the deploy added."

---

## Summary — what P1.1, once approved, actually needs to do

Given the above, P1.1's honest scope is:

1. Stand up versioned migration tooling (blocking dependency, §9).
2. Migration `000X`: `organizations` table + nullable `organization_id` on
   `users`/`departments`/`projects` + backfill to one default org +
   consistency check.
3. Migration `000X+1`: tighten to NOT NULL + FKs + the unique constraints
   listed in §9/§6, including the `users.email` scope decision (needs your
   confirmation — global-unique vs. per-org-unique).
4. `Organization` Go model + minimal repository (no API surface yet — that's
   Phase 19/P1.6+ per the brief's own phasing).
5. Nothing in P1.1 changes any authorization behavior yet (that's P1.2/P1.3)
   — existing endpoints keep working exactly as today, now with an
   (unused-so-far) `organization_id` populated underneath them.

Stopping here. Waiting for your approval before any code or migration is
written.
