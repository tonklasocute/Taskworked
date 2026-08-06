# Taskworked

Enterprise task management platform (Jira/Trello/Notion/ClickUp/Habitica-style),
internal tool. See [docs/superpowers/specs/2026-08-06-system-architecture-design.md](docs/superpowers/specs/2026-08-06-system-architecture-design.md)
for the architecture.

Stack: React + TypeScript + Vite + Tailwind (frontend), Go + Fiber + GORM
(backend), PostgreSQL, Redis, MinIO, Docker Compose.

## Run everything with Docker Compose

```bash
cp .env.example .env   # then fill in the secrets (JWT, Postgres, MinIO)
docker compose up --build
```

- Frontend: http://localhost
- API: http://localhost/api/v1 (proxied) or http://localhost:8080 directly
- MinIO console: http://localhost:9001

## Local development (without Docker for app code)

Start just the infra:

```bash
cp .env.example .env
docker compose up postgres redis minio
```

Backend:

```bash
cd backend
cp .env.example .env   # then fill in secrets
go run ./cmd/api
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

## Current status

Implemented: Auth module (register/login/refresh/logout, JWT + Redis
refresh tokens, RBAC middleware) end to end, backend and frontend
scaffolds, Docker Compose stack.

Not yet built: Projects, Tasks, Kanban, Calendar, Gantt, Action Plan,
Reports, Team, Notification, Gamification, AI Assistant — built
module-by-module per the architecture doc, core first.
