# Frontend UX/UI Overhaul — Enterprise Readiness

## Problem

The frontend works end-to-end but reads as a prototype:

- All 12 pages independently render `<div className="min-h-screen p-6">` with their own header/back-link/action buttons — no shared navigation shell.
- The 6 project-scoped views (List, Board, Calendar, Gantt, Action Plan, Reports) are unlinked sibling routes with no way to switch between them except editing the URL.
- The component library is limited to `Button` and `Card`; everything else (status pills, avatars, form fields, modals, menus, loading/empty states) is ad-hoc markup repeated per page.
- Dark-mode CSS tokens exist in `index.css` but nothing ever applies the `.dark` class — dead code today.
- No mutation anywhere surfaces an error to the user. A failed create/update/delete fails silently.

## Goals

Make the app read and behave like an internal enterprise tool: consistent navigation, a small reusable component set, visible feedback on every action, and a working dark mode — without changing any backend behavior or business logic.

## Non-Goals

- No new npm dependency (no Radix, no shadcn CLI, no toast library). Everything is hand-rolled Tailwind + `class-variance-authority`, matching the existing `button.tsx`/`card.tsx` pattern. `framer-motion` is already installed and covers animation needs (toast/dialog transitions).
- FullCalendar, the Gantt view, and Recharts internals are not rebuilt — only their surrounding chrome (headers, tabs, tokens) changes.
- No mobile-first redesign. This stays a desktop-first internal tool; the sidebar collapses to icon-only on narrow widths for basic usability, nothing more.
- No backend/API changes.
- No change to data-fetching logic, routes' business behavior, or existing tests' assertions beyond what's needed to keep them passing against new markup.

## Design

### 1. App Shell & Navigation

**`AppShell`** (replaces the current `AppLayout`, which today only holds a notification-bell strip):
- Persistent left sidebar: Dashboard, Projects, Team, Gamification, AI Assistant, Notification Settings. Collapses to icon-only; collapsed state persisted in `localStorage`.
- Slim top bar: breadcrumb slot, dark-mode toggle, notification bell (existing `NotificationBell`), user avatar/menu (Settings, Sign out) via the new `DropdownMenu` primitive.
- Main content renders through `<Outlet/>`.

**`ProjectLayout`** (new nested route): wraps the 6 project-view pages. Renders the project name + a `Tabs` bar (List/Board/Calendar/Gantt/Action Plan/Reports) once. The 6 pages drop their own duplicated "back to project" header and render only their content via `<Outlet/>`.

**Route changes in `AppRouter.tsx`:** the 6 `/projects/:projectId/*` routes move one level deeper, under a `ProjectLayout` element nested inside the existing `AppLayout`/`ProtectedRoute` wrapping.

### 2. Design System Components (`src/components/ui/`)

New files, each scoped and sized like the existing `button.tsx` (~30–50 lines):

| Component | Replaces |
|---|---|
| `badge.tsx` | Plain-text status/priority strings across Task/Project/Report pages |
| `avatar.tsx` | Bare name text for assignee/reporter/comment author |
| `input.tsx`, `select.tsx`, `textarea.tsx` | Raw `<input>`/`<select>`/`<textarea>` elements with ad-hoc classes (36 current usages) |
| `dialog.tsx` | One-off modal markup duplicated in `TaskDetailModal.tsx`, `TaskListPage.tsx`, `KanbanBoard.tsx` |
| `dropdown-menu.tsx` | User menu, sidebar overflow actions, per-row actions |
| `tabs.tsx` | Project sub-navigation (Section 1) |
| `skeleton.tsx` | The `"…"` loading placeholders used throughout |
| `empty-state.tsx` | Ad-hoc "No projects yet." one-liners |
| `toast.tsx` | New — see Section 3 |

`TaskDetailModal` and the two other one-off modals are refactored onto the new `Dialog` primitive as part of this work — markup changes, behavior does not.

### 3. Toast / Global Error Handling & Dark Mode

**Toast system:**
- `src/stores/toast-store.ts` — a Zustand store matching the existing `auth-store.ts` pattern (`show(message, variant)`, auto-dismiss timer, list of active toasts).
- `<Toaster/>` mounted once at the app root (in `App.tsx`), fixed top-right stack, `success`/`error`/`info` variants.
- **Global error coverage, zero per-call-site changes:** `App.tsx`'s `QueryClient` gains a `MutationCache({ onError })` that fires an error toast for *any* failed mutation app-wide — reads the server message from the existing axios error response shape when present, falls back to a generic "Something went wrong." message otherwise. This alone fixes silent-failure everywhere at once.
- **Success toasts are opt-in**, added only at the mutations where confirmation clearly matters (create/delete task, create/delete project) — not blanket-applied, to avoid toast noise on high-frequency actions (e.g. drag-and-drop status changes, checklist toggles).

**Dark mode:**
- `src/hooks/use-theme.ts` — reads `localStorage`, falls back to `prefers-color-scheme`, toggles the `.dark` class on `document.documentElement`. The CSS tokens in `index.css` already support this; nothing currently drives it.
- Toggle control lives in the `AppShell` top bar.

### 4. Rollout

Every existing page — `DashboardPage`, `ProjectsPage`, `TaskListPage`, `KanbanBoard`, `CalendarPage`, `GanttPage`, `ActionPlanPage`, `ReportsPage`, `TeamPage`, `GamificationPage`, `AIAssistantPage`, `NotificationPreferencesPage` — is migrated onto the new shell and components:
- Drop the page's own `min-h-screen`/header/back-link markup (now owned by `AppShell`/`ProjectLayout`).
- Replace plain-text status/priority with `Badge`, bare names with `Avatar`, `"…"` loading text with `Skeleton`, ad-hoc "no data" text with `EmptyState`, raw form elements with `Input`/`Select`/`Textarea`.
- Data-fetching (`useQuery`/`useMutation` calls) and business logic are untouched; only presentation markup changes.
- Dark-mode chart colors (Recharts in `ReportsPage`, Gantt) get a pass only where trivial via existing CSS variables — not a chart redesign.

## Testing

- No backend changes, so no Go tests are affected.
- Frontend has no existing test suite (`npm run lint`/`tsc -b` are the only current gates) — this work is verified via `npm run build` (type-check + build clean), `npm run lint` (oxlint clean), and live manual verification in the browser against the running dev stack: navigate every route, toggle dark mode, trigger at least one failing mutation to confirm the error toast fires, confirm the project tab bar switches views correctly.
