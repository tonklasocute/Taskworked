# System Architecture Design — Taskworked

Date: 2026-08-06
Status: Approved

## Context

Taskworked is an internal enterprise task management platform for a single
company (not multi-tenant SaaS), combining project/task management (Jira,
Trello, ClickUp), docs-style flexibility (Notion), and gamification
(Habitica). Full module list: Dashboard, Projects, Tasks, Kanban, Calendar,
Gantt, Action Plan, Reports, Team, Notification, Gamification, AI Assistant,
plus Auth/RBAC (5 roles: Super Admin, Admin, Manager, Leader, Employee).

This document is step 1 of the project's prescribed build order (System
Architecture → Folder Structure → Database Design → Backend → Frontend →
Deployment). It covers the architecture for **all** modules so later
decisions don't paint us into a corner, but implementation still proceeds
module-by-module, core first (Auth, Projects, Tasks, Kanban, Dashboard),
then Calendar/Gantt/Reports/Team, then Notification/Gamification/AI Assistant.

## Decisions from scoping

- **Audience/scale**: single company, internal tool. No multi-tenancy.
- **Repo layout**: monorepo — `/frontend` and `/backend` in this repo, one
  root `docker-compose.yml`.
- **Hosting**: single server (VPS/on-prem) via Docker Compose + Nginx. Not
  designing for k8s/multi-node from day one, but realtime/notification
  design (below) doesn't block adding replicas later.
- **AI provider**: Claude API (Anthropic), via a backend AI service layer.

## Architecture style

**Modular monolith**, not microservices. Single-server deployment and one
internal team mean microservices would add network hops, deployment
complexity, and distributed-transaction risk for no real benefit. Clean
Architecture + Repository Pattern + Service Layer (already mandated in
`PROJECT_RULE.md`) gives the same separation of concerns inside one
deployable. Any module can be peeled out into its own service later if it
needs independent scaling — the AI Assistant module is the most likely
candidate, given different latency/cost characteristics than the rest.

## Top-level components

```
┌─────────────┐      ┌───────────┐      ┌─────────────────────────┐
│ React SPA   │──────▶  Nginx    │──────▶  Go Fiber API (monolith)│
│ (Vite build)│      │ (reverse  │      │  - REST handlers        │
└─────────────┘      │  proxy +  │      │  - WebSocket hub        │
       ▲              │  static)  │      │  - Cron scheduler       │
       │ WS           └───────────┘      │  - AI service (Claude)  │
       └──────────────────────────────────▶ Notification service  │
                                          └─────────┬───────────────┘
                                                     │
                              ┌──────────────────────┼───────────────────┐
                              ▼                      ▼                   ▼
                        ┌──────────┐           ┌──────────┐        ┌──────────┐
                        │PostgreSQL│           │  Redis   │        │  MinIO   │
                        │ (system  │           │ (cache,  │        │(attach-  │
                        │  of      │           │ sessions,│        │ ments,   │
                        │  record) │           │ pub/sub) │        │ avatars) │
                        └──────────┘           └──────────┘        └──────────┘
```

One Go binary serves REST + WebSocket and runs the cron scheduler
in-process (`robfig/cron`). Redis is the JWT refresh-token/session store
and pub/sub bus for WebSocket fan-out (see Realtime, below — this is what
lets the API scale to multiple replicas later without a rewrite). MinIO is
S3-compatible object storage for all file attachments and avatars.

## Backend internal layering

Feature-based folders, one shape per module:

```
/backend
  /cmd/api            → main.go, wiring (DI root, plain constructor injection)
  /internal
    /modules
      /auth
        handler.go     → HTTP layer: parse request → call service → map response
        service.go     → business logic, orchestrates repo(s)
        repository.go  → GORM queries only, no business logic
        dto.go         → request/response structs + validation tags
        model.go       → GORM entity
      /project/...      (same shape)
      /task/...
      /kanban/...
      /calendar/...
      /gantt/...
      /report/...
      /team/...
      /notification/...
      /gamification/...
      /ai/...
    /middleware         → JWT auth, RBAC guard, request logging, rate limit
    /pkg                → shared: response envelope, errors, pagination
  /migrations
```

Each module is `handler → service → repository`, wired by plain
constructor injection in `cmd/api/main.go` — a DI framework (e.g. `wire`)
would be over-engineering at this team size. Modules only call each
other's `service` interfaces, never each other's repositories directly, so
a module's internals can change without breaking others.

## Auth & RBAC

- JWT access token (~15 min) + refresh token (~7 days), rotated on use.
  Refresh tokens stored in Redis keyed by user, so "logout everywhere" is a
  Redis delete.
- 5 org-level roles (Super Admin, Admin, Manager, Leader, Employee) as an
  enum column on `users`, enforced via a `RequireRole(...)` Fiber
  middleware per route group.
- Project-level membership is separate from org role: a `project_members`
  table with its own per-project role, since e.g. an Employee can still
  own a project they created.
- Forgot Password: emailed single-use reset token stored in Redis with a
  TTL.

## Realtime (Kanban, notifications, presence)

WebSocket hub inside the same Go process. Clients connect once (`/ws`),
authenticate with the JWT they already hold, and subscribe to channels
they have access to (`project:<id>`, `user:<id>`). Task/Kanban mutations
publish an event to Redis pub/sub after the DB write commits; the hub
relays it to connected clients. Using Redis pub/sub instead of an
in-memory broadcast means the design is already correct if the API ever
runs as 2+ replicas — no rewrite needed later.

## Notifications & Cron

A `notification` service with one interface and three backends: WebSocket
(realtime), Email (SMTP), LINE Notify (webhook). Callers (task assignment,
mention, overdue check) call `notifier.Send(userID, event)` without
knowing which channels fire — channel preference is a per-user settings
row. Scheduled digests (08:00 daily, 18:00 end-of-day, weekly, monthly)
and the overdue/near-due sweep are `robfig/cron` jobs registered at
startup in the same binary — no separate worker process needed at this
scale.

## Gamification

EXP/badge rules are event-driven, not hardcoded into task logic: task
completion emits a `TaskCompletedEvent`, and the gamification service
listens and applies the rule table (+20 base, +50 early, +100 critical,
-20 late, badges, streaks). This keeps gamification changeable/disable-able
without touching the `task` module.

## AI Assistant

Backend `ai` module wraps the Claude API (Anthropic) behind a service
interface (`GenerateTasks`, `SummarizeDaily`, `SummarizeWeekly`,
`EstimateDuration`, `SuggestPriority`, `SuggestAssignee`,
`PredictLateTasks`, `AnalyzeProductivity`, `GenerateMeetingSummary`). Kept
as its own module so it can be split into a separate service later if its
latency/cost profile ends up needing independent scaling or a job queue.

## Deployment

One `docker-compose.yml` at repo root:

- `postgres`, `redis`, `minio`
- `api` — the Go binary
- `web` — static Vite build served by `nginx`
- `nginx` — reverse proxy + TLS termination; routes `/api` and `/ws` to
  `api`, everything else to `web`

All configuration via environment variables (`.env`), never hardcoded —
per `PROJECT_RULE.md`. Migrations run as a one-shot `migrate` job before
`api` starts.

## Testing strategy

- Unit tests at the service layer with repository interfaces mocked — no
  DB required, this is where Clean Architecture pays off.
- A smaller set of repository tests against a real Postgres (via
  `testcontainers-go` or a docker-compose test profile) to catch query
  bugs.
- No architecture-level need for e2e tests yet; left as a later
  frontend/QA decision.

## Out of scope for this document

- Detailed folder structure for the frontend (step 2 of the build order).
- Database schema / ER diagram (step 3).
- Concrete API contracts (step 4, Backend).
- Multi-tenancy — explicitly not needed per scoping decision above; if
  this ever changes, it's a separate architecture revision, not an
  incremental patch.
