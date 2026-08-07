# Frontend UX/UI Enterprise-Readiness Overhaul Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the frontend's per-page duplicated headers, ad-hoc markup, and silent mutation failures with a shared navigation shell, a small hand-rolled design-system component set, working dark mode, and app-wide toast/error feedback — without touching any backend code or business logic.

**Architecture:** A new `AppShell` (sidebar + top bar) wraps every authenticated route; a new `ProjectLayout` adds a tab bar for the 6 project-scoped views. Both are built once, then every existing page is migrated onto them plus a set of new `src/components/ui/*` primitives (Badge, Avatar, Input, Select, Textarea, Dialog, DropdownMenu, Tabs, Skeleton, EmptyState, Toast) that replace repeated raw markup.

**Tech Stack:** React 19, TypeScript, Tailwind CSS 4, `class-variance-authority`, TanStack Query 5, Zustand, react-router-dom 7, lucide-react, framer-motion (all already installed — no new dependency added by this plan).

## Global Constraints

- No new npm dependency — every new component is hand-rolled Tailwind + `class-variance-authority`, matching `src/components/ui/button.tsx` and `card.tsx`.
- No backend/API changes, no change to `useQuery`/`useMutation` call signatures or query keys — only presentation markup changes.
- No test framework exists in this project (`package.json` has no Jest/Vitest/Playwright) and adding one is out of scope for this plan. Every task's verification step is `npm run build` (runs `tsc -b` then `vite build` — this is the project's only type-check gate), `npm run lint` (oxlint), and a concrete manual browser check against the dev server (`npm run dev`, backend + infra already running per `make dev` / `make infra`).
- Backend error envelope (confirmed from `backend/internal/pkg/response/response.go`): `{"success": bool, "data"?: any, "error"?: string}`. The global toast error handler (Task 1) reads `error.response.data.error`.
- All work happens in `frontend/` — paths below are relative to `/Users/cms-rd-1/Documents/GitHub/Taskworked/frontend` unless stated otherwise.

## File Structure

New files:
- `src/stores/toast-store.ts` — Zustand store holding active toasts.
- `src/components/ui/toast.tsx` — `<Toaster/>` renderer, mounted once in `App.tsx`.
- `src/hooks/use-theme.ts` — dark-mode state, drives `.dark` class on `<html>`.
- `src/components/ui/badge.tsx`, `avatar.tsx`, `input.tsx`, `textarea.tsx`, `select.tsx`, `skeleton.tsx`, `empty-state.tsx`, `dialog.tsx`, `dropdown-menu.tsx`, `tabs.tsx` — design-system primitives.
- `src/routes/ProjectLayout.tsx` — project header + view tab bar, wraps the 6 project-scoped pages.

Modified files:
- `src/routes/AppLayout.tsx` — rewritten in place (default export renamed `AppShell`, file path unchanged to minimize import churn) to add the sidebar/top bar.
- `src/routes/AppRouter.tsx` — nests the 6 `/projects/:projectId/*` routes under `ProjectLayout`.
- `src/App.tsx` — `QueryClient` gains a `MutationCache` with a global `onError` toast; mounts `<Toaster/>`.
- `index.html` — inline script to set `.dark` before paint (avoids a flash of the wrong theme).
- All 12 page components + `TaskDetailModal.tsx` — drop duplicated header markup, adopt the new primitives.

## Task 1: Toast store, global mutation error handling, dark mode

**Files:**
- Create: `src/stores/toast-store.ts`
- Create: `src/components/ui/toast.tsx`
- Create: `src/hooks/use-theme.ts`
- Modify: `src/App.tsx` (full rewrite, current content is 12 lines — see below)
- Modify: `index.html:6-8`

**Interfaces:**
- Produces: `useToastStore` (Zustand hook) with `show(message: string, variant?: "success" | "error" | "info") => void` and `dismiss(id: number) => void`; `Toast` and `ToastVariant` types.
- Produces: `<Toaster/>` — renders active toasts, no props.
- Produces: `useTheme()` — returns `{ theme: "light" | "dark", toggleTheme: () => void }`. Later consumed by `AppShell` (Task 4).

- [ ] **Step 1: Create the toast store**

```ts
// src/stores/toast-store.ts
import { create } from "zustand";

export type ToastVariant = "success" | "error" | "info";

export interface Toast {
  id: number;
  message: string;
  variant: ToastVariant;
}

interface ToastState {
  toasts: Toast[];
  show: (message: string, variant?: ToastVariant) => void;
  dismiss: (id: number) => void;
}

let nextId = 0;

export const useToastStore = create<ToastState>((set, get) => ({
  toasts: [],
  show: (message, variant = "info") => {
    const id = nextId++;
    set((state) => ({ toasts: [...state.toasts, { id, message, variant }] }));
    setTimeout(() => get().dismiss(id), 5000);
  },
  dismiss: (id) => set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) })),
}));
```

- [ ] **Step 2: Create the Toaster component**

```tsx
// src/components/ui/toast.tsx
import { AnimatePresence, motion } from "framer-motion";
import { CheckCircle2, Info, X, XCircle } from "lucide-react";
import { useToastStore, type Toast, type ToastVariant } from "@/stores/toast-store";
import { cn } from "@/lib/utils";

const ICON: Record<ToastVariant, typeof CheckCircle2> = {
  success: CheckCircle2,
  error: XCircle,
  info: Info,
};

const COLOR: Record<ToastVariant, string> = {
  success: "text-primary",
  error: "text-destructive",
  info: "text-foreground",
};

function ToastItem({ toast }: { toast: Toast }) {
  const dismiss = useToastStore((s) => s.dismiss);
  const Icon = ICON[toast.variant];

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: -8, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      className="glass flex w-80 items-start gap-2 rounded-xl p-3 shadow-lg"
    >
      <Icon className={cn("mt-0.5 h-4 w-4 shrink-0", COLOR[toast.variant])} />
      <p className="flex-1 text-sm">{toast.message}</p>
      <button onClick={() => dismiss(toast.id)} className="text-muted-foreground hover:text-foreground" aria-label="Dismiss">
        <X className="h-4 w-4" />
      </button>
    </motion.div>
  );
}

export function Toaster() {
  const toasts = useToastStore((s) => s.toasts);

  return (
    <div className="pointer-events-none fixed right-4 top-4 z-100 flex flex-col gap-2">
      <AnimatePresence>
        {toasts.map((t) => (
          <div key={t.id} className="pointer-events-auto">
            <ToastItem toast={t} />
          </div>
        ))}
      </AnimatePresence>
    </div>
  );
}
```

- [ ] **Step 3: Create the theme hook**

```ts
// src/hooks/use-theme.ts
import { useEffect, useState } from "react";

type Theme = "light" | "dark";
const STORAGE_KEY = "taskworked-theme";

function getInitialTheme(): Theme {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark") return stored;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(getInitialTheme);

  useEffect(() => {
    document.documentElement.classList.toggle("dark", theme === "dark");
    localStorage.setItem(STORAGE_KEY, theme);
  }, [theme]);

  return { theme, toggleTheme: () => setTheme((t) => (t === "light" ? "dark" : "light")) };
}
```

- [ ] **Step 4: Prevent flash-of-wrong-theme on load**

`index.html` currently has no `<script>` in `<head>`. Add one immediately after the `<title>` line (`index.html:7`):

```html
    <title>Taskworked</title>
    <script>
      (function () {
        var stored = localStorage.getItem("taskworked-theme");
        var dark = stored ? stored === "dark" : window.matchMedia("(prefers-color-scheme: dark)").matches;
        if (dark) document.documentElement.classList.add("dark");
      })();
    </script>
```

- [ ] **Step 5: Wire the global mutation error toast into `App.tsx`**

Replace the full contents of `src/App.tsx`:

```tsx
import { MutationCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import axios from "axios";
import AppRouter from "@/routes/AppRouter";
import { Toaster } from "@/components/ui/toast";
import { useToastStore } from "@/stores/toast-store";

function extractErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error) && typeof error.response?.data?.error === "string") {
    return error.response.data.error;
  }
  return "Something went wrong. Please try again.";
}

const queryClient = new QueryClient({
  mutationCache: new MutationCache({
    onError: (error) => {
      useToastStore.getState().show(extractErrorMessage(error), "error");
    },
  }),
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppRouter />
      <Toaster />
    </QueryClientProvider>
  );
}
```

- [ ] **Step 6: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

Then start the stack (`make infra` in another terminal if not already running, `make dev`) and in the browser: log in, open devtools Network tab, throttle/force a mutation to fail (easiest: in `ProjectsPage`, submit "New Project" with the backend temporarily stopped, or trigger any 4xx e.g. create a project with an empty name via devtools by removing the `required` attribute) — confirm a red toast reading the server's error message appears top-right and auto-dismisses after ~5s.

- [ ] **Step 7: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/stores/toast-store.ts frontend/src/components/ui/toast.tsx frontend/src/hooks/use-theme.ts frontend/src/App.tsx frontend/index.html
git commit -m "$(cat <<'EOF'
Add global toast system, mutation error handling, and dark mode hook

Every failed mutation now surfaces an error toast automatically via
QueryClient's MutationCache, fixing silent-failure across the app.
useTheme drives the existing (previously unused) .dark CSS tokens.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 2: Core form/display primitives (Badge, Avatar, Input, Textarea, Select, Skeleton, EmptyState)

**Files:**
- Create: `src/components/ui/badge.tsx`
- Create: `src/components/ui/avatar.tsx`
- Create: `src/components/ui/input.tsx`
- Create: `src/components/ui/textarea.tsx`
- Create: `src/components/ui/select.tsx`
- Create: `src/components/ui/skeleton.tsx`
- Create: `src/components/ui/empty-state.tsx`

**Interfaces:**
- Produces: `Badge` (component) + `BadgeProps` (type, exports `variant?: "default" | "primary" | "destructive" | "outline"`) — consumed by every rollout task for priority/status pills.
- Produces: `Avatar({ name, src, className })` — initials fallback, consumed by `AppShell` (Task 4), `TaskDetailModal`, `TeamPage`, `GamificationPage`.
- Produces: `Input`, `Textarea` — drop-in replacements for raw `<input>`/`<textarea>`, forward all native props + `ref`.
- Produces: `Select` — wraps native `<select>` with a chevron icon, forwards all native props + `ref` + `children`.
- Produces: `Skeleton({ className })` — a pulsing placeholder block.
- Produces: `EmptyState({ icon?, title, description?, action?, className? })`.

- [ ] **Step 1: Badge**

```tsx
// src/components/ui/badge.tsx
import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva("inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium", {
  variants: {
    variant: {
      default: "bg-muted text-muted-foreground",
      primary: "bg-primary/15 text-primary",
      destructive: "bg-destructive/15 text-destructive",
      outline: "border border-border text-foreground",
    },
  },
  defaultVariants: { variant: "default" },
});

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
```

- [ ] **Step 2: Avatar**

```tsx
// src/components/ui/avatar.tsx
import { cn } from "@/lib/utils";

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  const first = parts[0]?.[0] ?? "";
  const last = parts.length > 1 ? parts[parts.length - 1][0] : "";
  return (first + last).toUpperCase();
}

export function Avatar({ name, src, className }: { name: string; src?: string | null; className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary/15 text-xs font-medium text-primary",
        className,
      )}
    >
      {src ? <img src={src} alt={name} className="h-full w-full object-cover" /> : initials(name)}
    </span>
  );
}
```

- [ ] **Step 3: Input and Textarea**

```tsx
// src/components/ui/input.tsx
import * as React from "react";
import { cn } from "@/lib/utils";

export const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  ({ className, ...props }, ref) => (
    <input
      ref={ref}
      className={cn(
        "h-10 w-full rounded-lg border border-border bg-transparent px-3 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50",
        className,
      )}
      {...props}
    />
  ),
);
Input.displayName = "Input";
```

```tsx
// src/components/ui/textarea.tsx
import * as React from "react";
import { cn } from "@/lib/utils";

export const Textarea = React.forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(
        "min-h-20 w-full rounded-lg border border-border bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50",
        className,
      )}
      {...props}
    />
  ),
);
Textarea.displayName = "Textarea";
```

- [ ] **Step 4: Select**

```tsx
// src/components/ui/select.tsx
import * as React from "react";
import { ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";

export const Select = React.forwardRef<HTMLSelectElement, React.SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, children, ...props }, ref) => (
    <div className="relative">
      <select
        ref={ref}
        className={cn(
          "h-10 w-full appearance-none rounded-lg border border-border bg-transparent px-3 pr-8 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50",
          className,
        )}
        {...props}
      >
        {children}
      </select>
      <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
    </div>
  ),
);
Select.displayName = "Select";
```

- [ ] **Step 5: Skeleton and EmptyState**

```tsx
// src/components/ui/skeleton.tsx
import { cn } from "@/lib/utils";

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn("animate-pulse rounded-md bg-muted", className)} />;
}
```

```tsx
// src/components/ui/empty-state.tsx
import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
}: {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col items-center gap-2 rounded-xl border border-dashed border-border py-12 text-center", className)}>
      {Icon && <Icon className="h-8 w-8 text-muted-foreground" />}
      <p className="text-sm font-medium">{title}</p>
      {description && <p className="max-w-sm text-sm text-muted-foreground">{description}</p>}
      {action}
    </div>
  );
}
```

- [ ] **Step 6: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0. (These components aren't imported anywhere yet, so this only confirms they type-check standalone.)

- [ ] **Step 7: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/components/ui/badge.tsx frontend/src/components/ui/avatar.tsx frontend/src/components/ui/input.tsx frontend/src/components/ui/textarea.tsx frontend/src/components/ui/select.tsx frontend/src/components/ui/skeleton.tsx frontend/src/components/ui/empty-state.tsx
git commit -m "$(cat <<'EOF'
Add Badge, Avatar, Input, Textarea, Select, Skeleton, EmptyState primitives

Design-system building blocks for the frontend UX overhaul, matching
the existing hand-rolled cva + Tailwind pattern from button.tsx/card.tsx.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 3: Dialog, DropdownMenu, Tabs primitives

**Files:**
- Create: `src/components/ui/dialog.tsx`
- Create: `src/components/ui/dropdown-menu.tsx`
- Create: `src/components/ui/tabs.tsx`

**Interfaces:**
- Produces: `Dialog({ onClose, children, className? })` — fixed overlay + centered panel, closes on backdrop click or Escape. `DialogHeader({ title, description?, onClose })` — title row with a close button. Consumed by `TaskDetailModal` (Task 8).
- Produces: `DropdownMenu({ trigger, children, align?, className? })` + `DropdownMenuItem` (a styled `<button>`) — native `<details>/<summary>` disclosure, same technique already used by `NotificationBell`. Consumed by `AppShell` (Task 4).
- Produces: `Tabs` (flex/gap/border wrapper `<div>`) + `TabButton({ active, ...props })` — a segmented button group. Consumed by `GamificationPage` (Task 16), `ReportsPage` (Task 14), `GanttPage` (Task 12) to replace their existing ad-hoc `<div className="flex gap-1 rounded-lg border p-1">` + `Button` pattern.

- [ ] **Step 1: Dialog**

```tsx
// src/components/ui/dialog.tsx
import * as React from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

export function Dialog({
  onClose,
  children,
  className,
}: {
  onClose: () => void;
  children: React.ReactNode;
  className?: string;
}) {
  React.useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div
        className={cn("glass max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl p-6", className)}
        onClick={(e) => e.stopPropagation()}
      >
        {children}
      </div>
    </div>
  );
}

export function DialogHeader({
  title,
  description,
  onClose,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  onClose: () => void;
}) {
  return (
    <div className="mb-4 flex items-start justify-between gap-4">
      <div>
        <h2 className="text-lg font-semibold">{title}</h2>
        {description && <p className="mt-1 text-sm text-muted-foreground">{description}</p>}
      </div>
      <Button size="sm" variant="ghost" onClick={onClose} aria-label="Close">
        <X className="h-4 w-4" />
      </Button>
    </div>
  );
}
```

- [ ] **Step 2: DropdownMenu**

```tsx
// src/components/ui/dropdown-menu.tsx
import * as React from "react";
import { cn } from "@/lib/utils";

export function DropdownMenu({
  trigger,
  children,
  align = "end",
  className,
}: {
  trigger: React.ReactNode;
  children: React.ReactNode;
  align?: "start" | "end";
  className?: string;
}) {
  return (
    <details className="group relative">
      <summary className="cursor-pointer list-none [&::-webkit-details-marker]:hidden">{trigger}</summary>
      <div
        className={cn(
          "glass absolute z-20 mt-2 min-w-40 rounded-xl p-1 shadow-lg",
          align === "end" ? "right-0" : "left-0",
          className,
        )}
      >
        {children}
      </div>
    </details>
  );
}

export function DropdownMenuItem({ className, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      type="button"
      className={cn("flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-muted", className)}
      {...props}
    />
  );
}
```

- [ ] **Step 3: Tabs**

```tsx
// src/components/ui/tabs.tsx
import * as React from "react";
import { cn } from "@/lib/utils";

export function Tabs({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("inline-flex gap-1 rounded-lg border border-border p-1", className)} {...props} />;
}

export function TabButton({
  active,
  className,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { active: boolean }) {
  return (
    <button
      type="button"
      className={cn(
        "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
        active ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground",
        className,
      )}
      {...props}
    />
  );
}
```

- [ ] **Step 4: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

- [ ] **Step 5: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/components/ui/dialog.tsx frontend/src/components/ui/dropdown-menu.tsx frontend/src/components/ui/tabs.tsx
git commit -m "$(cat <<'EOF'
Add Dialog, DropdownMenu, Tabs primitives

Dialog generalizes TaskDetailModal's overlay/panel chrome (and adds
Escape-to-close). DropdownMenu reuses the native <details> disclosure
pattern already proven in NotificationBell. Tabs replaces the ad-hoc
segmented-button-group markup duplicated across Reports/Gantt/Gamification.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 4: AppShell (sidebar + top bar)

**Files:**
- Modify: `src/routes/AppLayout.tsx` (full rewrite — current file is 19 lines, see conversation history / `git show HEAD:frontend/src/routes/AppLayout.tsx`)

**Interfaces:**
- Consumes: `useAuthStore` (`user`, `clear`) from `src/stores/auth-store.ts`; `logout` from `src/features/auth/api.ts`; `useTheme` from Task 1; `Avatar`, `DropdownMenu`/`DropdownMenuItem` from Tasks 2–3; existing `NotificationBell` (`src/features/notifications/NotificationBell.tsx`, unchanged).
- Produces: default export `AppShell` (same file path `src/routes/AppLayout.tsx` — Task 5 imports it under this alias). Renders `<Outlet/>` for page content.

- [ ] **Step 1: Rewrite `src/routes/AppLayout.tsx`**

```tsx
import { useEffect, useState } from "react";
import { Link, NavLink, Outlet, useNavigate } from "react-router-dom";
import {
  ChevronLeft,
  ChevronRight,
  FolderKanban,
  LayoutDashboard,
  LogOut,
  Moon,
  Sparkles,
  Sun,
  Trophy,
  Users,
} from "lucide-react";
import { useAuthStore } from "@/stores/auth-store";
import { logout } from "@/features/auth/api";
import { useTheme } from "@/hooks/use-theme";
import { Avatar } from "@/components/ui/avatar";
import { DropdownMenu, DropdownMenuItem } from "@/components/ui/dropdown-menu";
import NotificationBell from "@/features/notifications/NotificationBell";
import { cn } from "@/lib/utils";

const NAV_ITEMS = [
  { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { to: "/projects", label: "Projects", icon: FolderKanban },
  { to: "/team", label: "Team", icon: Users },
  { to: "/gamification", label: "Gamification", icon: Trophy },
  { to: "/ai-assistant", label: "AI Assistant", icon: Sparkles },
];

const SIDEBAR_STORAGE_KEY = "taskworked-sidebar-collapsed";

export default function AppShell() {
  const user = useAuthStore((s) => s.user);
  const clear = useAuthStore((s) => s.clear);
  const navigate = useNavigate();
  const { theme, toggleTheme } = useTheme();

  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(SIDEBAR_STORAGE_KEY) === "1");

  useEffect(() => {
    localStorage.setItem(SIDEBAR_STORAGE_KEY, collapsed ? "1" : "0");
  }, [collapsed]);

  async function handleSignOut() {
    await logout();
    clear();
    navigate("/login");
  }

  return (
    <div className="flex min-h-screen">
      <aside
        className={cn(
          "flex shrink-0 flex-col border-r border-border bg-card/50 transition-[width] print:hidden",
          collapsed ? "w-16" : "w-56",
        )}
      >
        <div className="flex h-14 items-center gap-2 border-b border-border px-4">
          <span className="text-lg font-semibold">{collapsed ? "T" : "Taskworked"}</span>
        </div>
        <nav className="flex flex-1 flex-col gap-1 p-2">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                  isActive ? "bg-primary/10 text-primary" : "text-muted-foreground hover:bg-muted hover:text-foreground",
                )
              }
            >
              <item.icon className="h-5 w-5 shrink-0" />
              {!collapsed && item.label}
            </NavLink>
          ))}
        </nav>
        <button
          onClick={() => setCollapsed((v) => !v)}
          aria-label="Toggle sidebar"
          className="flex items-center gap-3 border-t border-border px-3 py-3 text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          {collapsed ? <ChevronRight className="h-5 w-5" /> : <ChevronLeft className="h-5 w-5" />}
          {!collapsed && "Collapse"}
        </button>
      </aside>

      <div className="flex min-h-screen flex-1 flex-col">
        <header className="sticky top-0 z-30 flex h-14 items-center justify-end gap-2 border-b border-border bg-background/80 px-4 backdrop-blur print:hidden">
          <button
            onClick={toggleTheme}
            aria-label="Toggle dark mode"
            className="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-muted"
          >
            {theme === "dark" ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
          </button>
          <NotificationBell />
          <DropdownMenu
            trigger={
              <span className="flex h-9 w-9 items-center justify-center rounded-lg hover:bg-muted">
                <Avatar name={user?.name ?? "?"} src={user?.avatar_url} />
              </span>
            }
          >
            <div className="border-b border-border px-3 py-2">
              <p className="text-sm font-medium">{user?.name}</p>
              <p className="text-xs text-muted-foreground">{user?.email}</p>
            </div>
            <Link
              to="/settings/notifications"
              className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-muted"
            >
              Notification settings
            </Link>
            <DropdownMenuItem onClick={handleSignOut}>
              <LogOut className="h-4 w-4" /> Sign out
            </DropdownMenuItem>
          </DropdownMenu>
        </header>

        <main className="flex-1">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify build**

Run: `npm run build && npm run lint`
Expected: both exit 0. (`AppRouter.tsx` still imports the old `AppLayout` default export by its old name at this point — Task 5 updates that reference. The rewrite here only changes what the default export renders, not its name/path, so this compiles standalone.)

- [ ] **Step 3: Manual check**

`npm run dev`, log in, confirm: sidebar renders with 5 nav items + collapse toggle (state persists across reload), top bar shows dark-mode toggle + notification bell + avatar; clicking the avatar opens a menu with your name/email, "Notification settings", and "Sign out"; clicking the sun/moon icon flips the whole app's palette immediately; "Sign out" logs you out and redirects to `/login`.

- [ ] **Step 4: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/routes/AppLayout.tsx
git commit -m "$(cat <<'EOF'
Add persistent sidebar + top bar shell (AppShell)

Replaces the bare notification-bell strip with real navigation: a
collapsible sidebar (5 primary sections) and a top bar carrying the
dark-mode toggle, notifications, and a user menu with sign-out.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 5: ProjectLayout + route nesting

**Files:**
- Create: `src/routes/ProjectLayout.tsx`
- Modify: `src/routes/AppRouter.tsx` (full rewrite, current file is 49 lines)

**Interfaces:**
- Consumes: `getProject` from `src/features/projects/api.ts` (already used identically by every project-scoped page today).
- Produces: default export `ProjectLayout`, rendering project title + a 6-item tab bar (List/Board/Calendar/Gantt/Action Plan/Reports) then `<Outlet/>`. Nested under `/projects/:projectId` in the router; `TaskListPage` becomes the `index` route, the other 5 become relative children (`board`, `calendar`, `gantt`, `action-plan`, `reports`) — their own components (Tasks 9–14) no longer need `useParams` fallback links to sibling views since the tab bar now owns that navigation.

- [ ] **Step 1: Create `src/routes/ProjectLayout.tsx`**

```tsx
import { Link, NavLink, Outlet, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { getProject } from "@/features/projects/api";
import { cn } from "@/lib/utils";

const TABS = [
  { to: "", label: "List", end: true },
  { to: "board", label: "Board", end: false },
  { to: "calendar", label: "Calendar", end: false },
  { to: "gantt", label: "Gantt", end: false },
  { to: "action-plan", label: "Action Plan", end: false },
  { to: "reports", label: "Reports", end: false },
];

export default function ProjectLayout() {
  const { projectId } = useParams<{ projectId: string }>();

  const { data: project } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => getProject(projectId!),
    enabled: !!projectId,
  });

  return (
    <div className="flex min-h-screen flex-col">
      <div className="border-b border-border px-6 pt-6 print:hidden">
        <Link to="/projects" className="mb-2 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="h-4 w-4" /> Projects
        </Link>
        <h1 className="mb-4 text-2xl font-semibold">{project?.name ?? "Project"}</h1>
        <nav className="flex gap-1">
          {TABS.map((tab) => (
            <NavLink
              key={tab.label}
              to={tab.to}
              end={tab.end}
              className={({ isActive }) =>
                cn(
                  "rounded-t-lg px-3 py-2 text-sm font-medium",
                  isActive ? "border-b-2 border-primary text-primary" : "text-muted-foreground hover:text-foreground",
                )
              }
            >
              {tab.label}
            </NavLink>
          ))}
        </nav>
      </div>
      <div className="flex-1">
        <Outlet />
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Rewrite `src/routes/AppRouter.tsx`**

```tsx
import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import ProtectedRoute from "@/routes/ProtectedRoute";
import AppShell from "@/routes/AppLayout";
import ProjectLayout from "@/routes/ProjectLayout";
import LoginPage from "@/features/auth/LoginPage";
import RegisterPage from "@/features/auth/RegisterPage";
import DashboardPage from "@/features/dashboard/DashboardPage";
import ProjectsPage from "@/features/projects/ProjectsPage";
import TaskListPage from "@/features/tasks/TaskListPage";
import KanbanBoard from "@/features/tasks/KanbanBoard";
import CalendarPage from "@/features/tasks/CalendarPage";
import GanttPage from "@/features/tasks/GanttPage";
import ActionPlanPage from "@/features/actionplan/ActionPlanPage";
import ReportsPage from "@/features/reports/ReportsPage";
import TeamPage from "@/features/team/TeamPage";
import NotificationPreferencesPage from "@/features/notifications/NotificationPreferencesPage";
import GamificationPage from "@/features/gamification/GamificationPage";
import AIAssistantPage from "@/features/ai/AIAssistantPage";

const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  { path: "/register", element: <RegisterPage /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppShell />,
        children: [
          { path: "/dashboard", element: <DashboardPage /> },
          { path: "/projects", element: <ProjectsPage /> },
          {
            path: "/projects/:projectId",
            element: <ProjectLayout />,
            children: [
              { index: true, element: <TaskListPage /> },
              { path: "board", element: <KanbanBoard /> },
              { path: "calendar", element: <CalendarPage /> },
              { path: "gantt", element: <GanttPage /> },
              { path: "action-plan", element: <ActionPlanPage /> },
              { path: "reports", element: <ReportsPage /> },
            ],
          },
          { path: "/team", element: <TeamPage /> },
          { path: "/settings/notifications", element: <NotificationPreferencesPage /> },
          { path: "/gamification", element: <GamificationPage /> },
          { path: "/ai-assistant", element: <AIAssistantPage /> },
        ],
      },
    ],
  },
  { path: "/", element: <Navigate to="/dashboard" replace /> },
]);

export default function AppRouter() {
  return <RouterProvider router={router} />;
}
```

- [ ] **Step 3: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, navigate to a project (`/projects/:id`) — confirm the tab bar renders under the project title and switching tabs updates the URL (`/projects/:id/board`, `/projects/:id/calendar`, etc.) without a full page reload. Pages will look visually broken/duplicated at this point (they still render their own old headers too) — that's expected and fixed by Tasks 9–14; this task only proves the routing/tab-bar shell itself works.

- [ ] **Step 4: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/routes/ProjectLayout.tsx frontend/src/routes/AppRouter.tsx
git commit -m "$(cat <<'EOF'
Add ProjectLayout tab bar, nest project-scoped routes under it

List/Board/Calendar/Gantt/Action Plan/Reports were unlinked sibling
routes reachable only by editing the URL. ProjectLayout gives them a
persistent header + tab bar; the 6 pages are next migrated (Tasks 9-14)
to drop their own duplicated headers now that this owns navigation.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

**End of Phase A (foundation).** Every task from here on migrates one existing page onto the shell + primitives built above. Each is independently testable and can be committed/reviewed on its own.

## Task 6: Migrate DashboardPage

**Files:**
- Modify: `src/features/dashboard/DashboardPage.tsx` (full rewrite, current file is 165 lines)

**Interfaces:**
- Consumes: `Badge`/`BadgeProps` (Task 2), `Skeleton` (Task 2), `EmptyState` (Task 2). Sign-out button removed — now lives in `AppShell` (Task 4).

- [ ] **Step 1: Replace the full contents of `src/features/dashboard/DashboardPage.tsx`**

```tsx
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge, type BadgeProps } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import { useAuthStore } from "@/stores/auth-store";
import { getMySummary } from "@/features/tasks/api";
import { listProjects } from "@/features/projects/api";
import { getProfile } from "@/features/gamification/api";
import type { TaskPriority } from "@/features/tasks/types";

const PRIORITY_LABEL: Record<TaskPriority, string> = {
  critical: "Critical",
  high: "High",
  medium: "Medium",
  low: "Low",
};

const PRIORITY_VARIANT: Record<TaskPriority, BadgeProps["variant"]> = {
  critical: "destructive",
  high: "primary",
  medium: "default",
  low: "outline",
};

const LEVEL_EMOJI: Record<string, string> = {
  Novice: "🌱",
  Adept: "🌿",
  Master: "🌳",
  Legend: "🏆",
};

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);

  const { data: summary, isLoading: summaryLoading } = useQuery({ queryKey: ["my-summary"], queryFn: getMySummary });
  const { data: projects } = useQuery({ queryKey: ["projects-for-dashboard"], queryFn: listProjects });
  const { data: profile } = useQuery({ queryKey: ["gamification-profile"], queryFn: getProfile });

  const recentProjects = projects?.items.slice(0, 5) ?? [];

  return (
    <div className="p-6">
      <h1 className="mb-6 text-2xl font-semibold">Welcome back, {user?.name}</h1>

      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="p-5">
            <p className="text-xs text-muted-foreground">Active tasks</p>
            {summaryLoading ? (
              <Skeleton className="mt-1 h-9 w-12" />
            ) : (
              <p className="mt-1 text-3xl font-semibold">{summary?.active_count ?? 0}</p>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <p className="text-xs text-muted-foreground">Overdue</p>
            {summaryLoading ? (
              <Skeleton className="mt-1 h-9 w-12" />
            ) : (
              <p className={`mt-1 text-3xl font-semibold ${(summary?.overdue_count ?? 0) > 0 ? "text-destructive" : ""}`}>
                {summary?.overdue_count ?? 0}
              </p>
            )}
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-5">
            <p className="text-xs text-muted-foreground">Due within 3 days</p>
            {summaryLoading ? (
              <Skeleton className="mt-1 h-9 w-12" />
            ) : (
              <p className="mt-1 text-3xl font-semibold">{summary?.due_soon_count ?? 0}</p>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>My Tasks</CardTitle>
          </CardHeader>
          <CardContent>
            {summaryLoading && (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-14 w-full" />
                <Skeleton className="h-14 w-full" />
                <Skeleton className="h-14 w-full" />
              </div>
            )}
            {summary && summary.tasks.length === 0 && (
              <EmptyState title="No active tasks" description="Nothing assigned to you right now." />
            )}
            <ul className="flex flex-col gap-2">
              {summary?.tasks.map((t) => (
                <li key={t.id}>
                  <Link
                    to={`/projects/${t.project_id}`}
                    className="flex items-center justify-between rounded-lg border border-border p-3 text-sm hover:bg-muted"
                  >
                    <span className="flex flex-col gap-1">
                      <span className="font-medium">{t.title}</span>
                      <span className="flex items-center gap-2">
                        <Badge variant={PRIORITY_VARIANT[t.priority]}>{PRIORITY_LABEL[t.priority]}</Badge>
                        <span className="text-xs text-muted-foreground">{t.status}</span>
                      </span>
                    </span>
                    {t.due_date && <span className="text-xs text-muted-foreground">{t.due_date}</span>}
                  </Link>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>

        <div className="flex flex-col gap-6">
          {profile && (
            <Card>
              <CardHeader>
                <CardTitle>Your Progress</CardTitle>
              </CardHeader>
              <CardContent>
                <Link to="/gamification" className="flex items-center gap-3">
                  <span className="text-2xl">{LEVEL_EMOJI[profile.character.level_title] ?? "🌱"}</span>
                  <span>
                    <span className="block font-medium">
                      Level {profile.character.level} — {profile.character.level_title}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      🔥 {profile.character.current_streak} day streak · {profile.character.total_completed} completed
                    </span>
                  </span>
                </Link>
              </CardContent>
            </Card>
          )}

          <Card>
            <CardHeader>
              <CardTitle>Recent Projects</CardTitle>
            </CardHeader>
            <CardContent>
              {recentProjects.length === 0 && <EmptyState title="No projects yet" />}
              <ul className="flex flex-col gap-2">
                {recentProjects.map((p) => (
                  <li key={p.id}>
                    <Link to={`/projects/${p.id}`} className="block rounded-lg border border-border p-3 text-sm hover:bg-muted">
                      <span className="font-medium">{p.name}</span>
                      <span className="ml-2 text-xs text-muted-foreground">{p.status}</span>
                    </Link>
                  </li>
                ))}
              </ul>
              <Link to="/projects" className="mt-3 block text-xs text-muted-foreground hover:text-foreground">
                View all projects →
              </Link>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit `/dashboard`: confirm no duplicate header/nav buttons remain (sidebar covers that now), priority badges render with distinct colors (critical=red-tinted, high=purple-tinted, medium=gray, low=outlined), and briefly throttling network (devtools) shows skeleton blocks instead of "…" text while loading.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/dashboard/DashboardPage.tsx
git commit -m "$(cat <<'EOF'
Migrate DashboardPage onto the new shell and design-system primitives

Drops the page's own nav-button row (now the sidebar's job) and
sign-out button (now the top-bar user menu's job); swaps plain-text
priority labels for Badge and "…" loading text for Skeleton.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 7: Migrate ProjectsPage

**Files:**
- Modify: `src/features/projects/ProjectsPage.tsx` (full rewrite, current file is 105 lines)

**Interfaces:**
- Consumes: `Badge`/`BadgeProps`, `Input`, `Textarea`, `Skeleton`, `EmptyState` (Task 2).

- [ ] **Step 1: Replace the full contents of `src/features/projects/ProjectsPage.tsx`**

```tsx
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { createProject, listProjects } from "@/features/projects/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Badge, type BadgeProps } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";

const STATUS_LABEL: Record<string, string> = {
  planning: "Planning",
  active: "Active",
  on_hold: "On Hold",
  completed: "Completed",
  archived: "Archived",
};

const STATUS_VARIANT: Record<string, BadgeProps["variant"]> = {
  planning: "default",
  active: "primary",
  on_hold: "outline",
  completed: "primary",
  archived: "default",
};

export default function ProjectsPage() {
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const { data, isLoading } = useQuery({ queryKey: ["projects"], queryFn: listProjects });

  const mutation = useMutation({
    mutationFn: () => createProject({ name, description }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
      setName("");
      setDescription("");
      setShowForm(false);
    },
  });

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Projects</h1>
        <Button onClick={() => setShowForm((v) => !v)}>{showForm ? "Cancel" : "New Project"}</Button>
      </div>

      {showForm && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create project</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="flex flex-col gap-3"
              onSubmit={(e) => {
                e.preventDefault();
                mutation.mutate();
              }}
            >
              <Input required placeholder="Project name" value={name} onChange={(e) => setName(e.target.value)} />
              <Textarea
                placeholder="Description (optional)"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
              <Button type="submit" disabled={mutation.isPending} className="self-start">
                {mutation.isPending ? "Creating…" : "Create project"}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {isLoading && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-32 w-full" />
        </div>
      )}

      {!isLoading && data?.items.length === 0 && (
        <EmptyState title="No projects yet" description="Create your first one above." />
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {data?.items.map((project) => (
          <Link key={project.id} to={`/projects/${project.id}`}>
            <Card className="h-full transition-opacity hover:opacity-90">
              <CardHeader>
                <CardTitle>{project.name}</CardTitle>
              </CardHeader>
              <CardContent>
                {project.description && (
                  <p className="mb-3 line-clamp-2 text-sm text-muted-foreground">{project.description}</p>
                )}
                <Badge variant={STATUS_VARIANT[project.status]}>{STATUS_LABEL[project.status] ?? project.status}</Badge>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit `/projects`: confirm no "← Dashboard" link remains (sidebar covers it), status pills render as colored Badges, and the create-project form's input/textarea look identical in styling to before (same border/radius/padding — `Input`/`Textarea` were built to match the prior raw-element classes exactly).

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/projects/ProjectsPage.tsx
git commit -m "$(cat <<'EOF'
Migrate ProjectsPage onto the new shell and design-system primitives

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 8: Migrate TaskDetailModal onto Dialog

**Files:**
- Modify: `src/features/tasks/TaskDetailModal.tsx` (full rewrite, current file is 271 lines)

**Interfaces:**
- Consumes: `Dialog`, `DialogHeader` (Task 3); `Avatar` (Task 2); `Input` (Task 2). No change to props (`{ taskId: string; onClose: () => void }`), query keys, or mutation call signatures — `TaskListPage` (Task 9) and `KanbanBoard` (Task 10) render this unchanged.

- [ ] **Step 1: Replace the full contents of `src/features/tasks/TaskDetailModal.tsx`**

```tsx
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/auth-store";
import {
  addChecklistItem,
  addComment,
  deleteAttachment,
  deleteChecklistItem,
  deleteComment,
  getAttachmentDownloadURL,
  getTask,
  getWatchStatus,
  listAttachments,
  listChecklist,
  listComments,
  listWatchers,
  setTags,
  unwatchTask,
  updateChecklistItem,
  uploadAttachment,
  watchTask,
} from "@/features/tasks/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Avatar } from "@/components/ui/avatar";
import { Dialog, DialogHeader } from "@/components/ui/dialog";

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function TaskDetailModal({ taskId, onClose }: { taskId: string; onClose: () => void }) {
  const queryClient = useQueryClient();
  const currentUserId = useAuthStore((s) => s.user?.id);
  const [commentBody, setCommentBody] = useState("");
  const [newChecklistText, setNewChecklistText] = useState("");
  const [tagInput, setTagInput] = useState("");

  const { data: task } = useQuery({ queryKey: ["task", taskId], queryFn: () => getTask(taskId) });
  const { data: comments } = useQuery({ queryKey: ["task-comments", taskId], queryFn: () => listComments(taskId) });
  const { data: attachments } = useQuery({ queryKey: ["task-attachments", taskId], queryFn: () => listAttachments(taskId) });
  const { data: watchers } = useQuery({ queryKey: ["task-watchers", taskId], queryFn: () => listWatchers(taskId) });
  const { data: watchStatus } = useQuery({ queryKey: ["task-watch-status", taskId], queryFn: () => getWatchStatus(taskId) });
  const { data: checklist } = useQuery({ queryKey: ["task-checklist", taskId], queryFn: () => listChecklist(taskId) });

  const invalidateChecklist = () => queryClient.invalidateQueries({ queryKey: ["task-checklist", taskId] });
  const addChecklistMutation = useMutation({
    mutationFn: (text: string) => addChecklistItem(taskId, text),
    onSuccess: () => {
      setNewChecklistText("");
      invalidateChecklist();
    },
  });
  const toggleChecklistMutation = useMutation({
    mutationFn: ({ id, done }: { id: string; done: boolean }) => updateChecklistItem(taskId, id, { done }),
    onSuccess: invalidateChecklist,
  });
  const deleteChecklistMutation = useMutation({
    mutationFn: (id: string) => deleteChecklistItem(taskId, id),
    onSuccess: invalidateChecklist,
  });

  const invalidateTask = () => queryClient.invalidateQueries({ queryKey: ["task", taskId] });
  const setTagsMutation = useMutation({ mutationFn: (tags: string[]) => setTags(taskId, tags), onSuccess: invalidateTask });

  const invalidateComments = () => queryClient.invalidateQueries({ queryKey: ["task-comments", taskId] });
  const addCommentMutation = useMutation({
    mutationFn: () => addComment(taskId, commentBody),
    onSuccess: () => {
      setCommentBody("");
      invalidateComments();
    },
  });
  const deleteCommentMutation = useMutation({ mutationFn: (id: string) => deleteComment(taskId, id), onSuccess: invalidateComments });

  const invalidateAttachments = () => queryClient.invalidateQueries({ queryKey: ["task-attachments", taskId] });
  const uploadMutation = useMutation({ mutationFn: (file: File) => uploadAttachment(taskId, file), onSuccess: invalidateAttachments });
  const deleteAttachmentMutation = useMutation({
    mutationFn: (id: string) => deleteAttachment(taskId, id),
    onSuccess: invalidateAttachments,
  });
  const downloadMutation = useMutation({
    mutationFn: (id: string) => getAttachmentDownloadURL(taskId, id),
    onSuccess: (url) => window.open(url, "_blank"),
  });

  const invalidateWatch = () => {
    queryClient.invalidateQueries({ queryKey: ["task-watchers", taskId] });
    queryClient.invalidateQueries({ queryKey: ["task-watch-status", taskId] });
  };
  const watchMutation = useMutation({ mutationFn: () => watchTask(taskId), onSuccess: invalidateWatch });
  const unwatchMutation = useMutation({ mutationFn: () => unwatchTask(taskId), onSuccess: invalidateWatch });

  return (
    <Dialog onClose={onClose}>
      <DialogHeader title={task?.title ?? "Loading…"} description={task?.description} onClose={onClose} />

      <div className="mb-4 flex flex-wrap items-center gap-2">
        {task?.tags.map((tag) => (
          <span key={tag} className="flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
            {tag}
            <button
              className="hover:text-destructive"
              onClick={() => setTagsMutation.mutate((task?.tags ?? []).filter((t) => t !== tag))}
            >
              ✕
            </button>
          </span>
        ))}
        <form
          className="inline-flex"
          onSubmit={(e) => {
            e.preventDefault();
            const t = tagInput.trim();
            if (t && !task?.tags.includes(t)) {
              setTagsMutation.mutate([...(task?.tags ?? []), t]);
              setTagInput("");
            }
          }}
        >
          <Input
            placeholder="+ tag"
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
            className="h-6 w-20 rounded-full px-2 text-xs"
          />
        </form>
      </div>

      <div className="mb-6 flex items-center gap-3">
        <Button
          size="sm"
          variant={watchStatus?.watching ? "default" : "outline"}
          onClick={() => (watchStatus?.watching ? unwatchMutation.mutate() : watchMutation.mutate())}
        >
          {watchStatus?.watching ? "★ Watching" : "☆ Watch"}
        </Button>
        <span className="text-xs text-muted-foreground">
          {watchers?.length ?? 0} watching
          {watchers && watchers.length > 0 ? `: ${watchers.map((w) => w.name).join(", ")}` : ""}
        </span>
      </div>

      <section className="mb-6">
        <h3 className="mb-2 text-sm font-semibold">Checklist</h3>
        <ul className="mb-3 flex flex-col gap-1">
          {checklist?.map((item) => (
            <li key={item.id} className="flex items-center gap-2 rounded-lg px-2 py-1 text-sm hover:bg-muted">
              <input
                type="checkbox"
                checked={item.done}
                onChange={(e) => toggleChecklistMutation.mutate({ id: item.id, done: e.target.checked })}
              />
              <span className={`flex-1 ${item.done ? "text-muted-foreground line-through" : ""}`}>{item.text}</span>
              <button
                className="text-xs text-muted-foreground hover:text-destructive"
                onClick={() => deleteChecklistMutation.mutate(item.id)}
              >
                ✕
              </button>
            </li>
          ))}
          {checklist?.length === 0 && <p className="text-sm text-muted-foreground">No checklist items yet.</p>}
        </ul>
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (newChecklistText.trim()) addChecklistMutation.mutate(newChecklistText);
          }}
        >
          <Input
            placeholder="Add a checklist item…"
            value={newChecklistText}
            onChange={(e) => setNewChecklistText(e.target.value)}
          />
          <Button type="submit" size="sm" disabled={addChecklistMutation.isPending}>
            Add
          </Button>
        </form>
      </section>

      <section className="mb-6">
        <h3 className="mb-2 text-sm font-semibold">Attachments</h3>
        <ul className="mb-3 flex flex-col gap-2">
          {attachments?.map((a) => (
            <li key={a.id} className="flex items-center justify-between rounded-lg border border-border p-2 text-sm">
              <button className="text-left hover:underline" onClick={() => downloadMutation.mutate(a.id)}>
                {a.file_name} <span className="text-xs text-muted-foreground">({formatSize(a.size_bytes)})</span>
              </button>
              {a.uploader_id === currentUserId && (
                <Button size="sm" variant="ghost" onClick={() => deleteAttachmentMutation.mutate(a.id)}>
                  Delete
                </Button>
              )}
            </li>
          ))}
          {attachments?.length === 0 && <p className="text-sm text-muted-foreground">No attachments yet.</p>}
        </ul>
        <input
          type="file"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) uploadMutation.mutate(file);
            e.target.value = "";
          }}
          disabled={uploadMutation.isPending}
          className="text-sm"
        />
      </section>

      <section>
        <h3 className="mb-2 text-sm font-semibold">Comments</h3>
        <ul className="mb-3 flex flex-col gap-3">
          {comments?.map((c) => (
            <li key={c.id} className="flex gap-3 rounded-lg border border-border p-3 text-sm">
              <Avatar name={c.author_name} className="mt-0.5" />
              <div className="flex-1">
                <div className="mb-1 flex items-center justify-between">
                  <span className="font-medium">{c.author_name}</span>
                  <span className="text-xs text-muted-foreground">{new Date(c.created_at).toLocaleString()}</span>
                </div>
                <p className="whitespace-pre-wrap">{c.body}</p>
                {c.author_id === currentUserId && (
                  <button
                    className="mt-1 text-xs text-muted-foreground hover:text-destructive"
                    onClick={() => deleteCommentMutation.mutate(c.id)}
                  >
                    Delete
                  </button>
                )}
              </div>
            </li>
          ))}
          {comments?.length === 0 && <p className="text-sm text-muted-foreground">No comments yet.</p>}
        </ul>
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (commentBody.trim()) addCommentMutation.mutate();
          }}
        >
          <Input placeholder="Write a comment…" value={commentBody} onChange={(e) => setCommentBody(e.target.value)} />
          <Button type="submit" size="sm" disabled={addCommentMutation.isPending}>
            Post
          </Button>
        </form>
      </section>
    </Dialog>
  );
}
```

Note: the old per-attachment-upload inline error text (`Upload failed. Is object storage configured?`) is intentionally dropped here — `uploadMutation`'s failure now surfaces through the app-wide `MutationCache` error toast from Task 1, so a second, page-local error message would be redundant.

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, open any task's detail modal from the List or Board view: confirm it still opens/closes the same way (backdrop click, and now also the Escape key), tags/checklist/attachments/comments/watch all still work exactly as before, and the close button now shows an X icon instead of a "✕" glyph.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/tasks/TaskDetailModal.tsx
git commit -m "$(cat <<'EOF'
Migrate TaskDetailModal onto the Dialog primitive

Adds Escape-to-close for free. Comment authors now show an Avatar.
Upload failures rely on the app-wide error toast instead of a
page-local message, avoiding duplicate error UI.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 9: Migrate TaskListPage

**Files:**
- Modify: `src/features/tasks/TaskListPage.tsx` (full rewrite, current file is 162 lines)

**Interfaces:**
- Consumes: `Badge`/`BadgeProps`, `Input`, `Select` (Task 2). `TaskDetailModal` (Task 8, unchanged props). The page no longer renders links to sibling views (Board/Calendar/Gantt/Action Plan/Reports) or a "← Projects"/project-name header — `ProjectLayout` (Task 5) now owns all of that via its tab bar.

- [ ] **Step 1: Replace the full contents of `src/features/tasks/TaskListPage.tsx`**

```tsx
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { createTask, listTasks, updateTaskStatus } from "@/features/tasks/api";
import type { TaskPriority, TaskStatus } from "@/features/tasks/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Badge, type BadgeProps } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/ui/empty-state";
import TaskDetailModal from "@/features/tasks/TaskDetailModal";

const STATUSES: TaskStatus[] = ["backlog", "todo", "doing", "review", "testing", "done", "blocked"];
const STATUS_LABEL: Record<TaskStatus, string> = {
  backlog: "Backlog",
  todo: "Todo",
  doing: "Doing",
  review: "Review",
  testing: "Testing",
  done: "Done",
  blocked: "Blocked",
};
const PRIORITY_VARIANT: Record<TaskPriority, BadgeProps["variant"]> = {
  critical: "destructive",
  high: "primary",
  medium: "default",
  low: "outline",
};

export default function TaskListPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [title, setTitle] = useState("");
  const [priority, setPriority] = useState<TaskPriority>("medium");
  const [openTaskId, setOpenTaskId] = useState<string | null>(null);

  const { data: tasks, isLoading } = useQuery({
    queryKey: ["tasks", projectId],
    queryFn: () => listTasks(projectId!),
    enabled: !!projectId,
  });

  const createMutation = useMutation({
    mutationFn: () => createTask({ project_id: projectId!, title, priority }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] });
      setTitle("");
      setShowForm(false);
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: TaskStatus }) => updateTaskStatus(id, status),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["tasks", projectId] }),
  });

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-end">
        <Button onClick={() => setShowForm((v) => !v)}>{showForm ? "Cancel" : "New Task"}</Button>
      </div>

      {showForm && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create task</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="flex flex-col gap-3 sm:flex-row sm:items-end"
              onSubmit={(e) => {
                e.preventDefault();
                createMutation.mutate();
              }}
            >
              <Input
                required
                placeholder="Task title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="flex-1"
              />
              <Select value={priority} onChange={(e) => setPriority(e.target.value as TaskPriority)} className="sm:w-40">
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
              </Select>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? "Creating…" : "Create task"}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {isLoading && (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      )}

      {!isLoading && tasks?.items.length === 0 && (
        <EmptyState title="No tasks yet" description="Create your first one above." />
      )}

      <div className="flex flex-col gap-3">
        {tasks?.items.map((t) => (
          <Card key={t.id}>
            <CardContent className="flex flex-wrap items-center justify-between gap-3 py-4">
              <button className="text-left" onClick={() => setOpenTaskId(t.id)}>
                <p className="font-medium hover:underline">{t.title}</p>
                <Badge variant={PRIORITY_VARIANT[t.priority]} className="mt-1">
                  {t.priority}
                </Badge>
              </button>
              <Select
                value={t.status}
                onChange={(e) => statusMutation.mutate({ id: t.id, status: e.target.value as TaskStatus })}
                className="h-9 w-40"
              >
                {STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {STATUS_LABEL[s]}
                  </option>
                ))}
              </Select>
            </CardContent>
          </Card>
        ))}
      </div>

      {openTaskId && <TaskDetailModal taskId={openTaskId} onClose={() => setOpenTaskId(null)} />}
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit a project's List tab: confirm the project title/tab bar (from `ProjectLayout`) shows above the page, no duplicate "← Projects" link or view-switcher buttons remain here, priority shows as a Badge, and status changes via the dropdown still update immediately.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/tasks/TaskListPage.tsx
git commit -m "$(cat <<'EOF'
Migrate TaskListPage onto ProjectLayout and design-system primitives

Drops the page's own project header and view-switcher button row,
now owned by ProjectLayout's tab bar.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 10: Migrate KanbanBoard

**Files:**
- Modify: `src/features/tasks/KanbanBoard.tsx` (full rewrite, current file is 152 lines)

**Interfaces:**
- Consumes: `Badge`/`BadgeProps` (Task 2). `TaskDetailModal` (Task 8, unchanged props). Drops its own "← List view" header — owned by `ProjectLayout` now. `dnd-kit` drag logic (`DndContext`, `useDraggable`, `useDroppable`) is untouched.

- [ ] **Step 1: Replace the full contents of `src/features/tasks/KanbanBoard.tsx`**

```tsx
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { DndContext, useDraggable, useDroppable, type DragEndEvent } from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import { listTasks, updateTaskStatus } from "@/features/tasks/api";
import { useTaskSocket } from "@/features/tasks/useTaskSocket";
import type { Task, TaskPriority, TaskStatus } from "@/features/tasks/types";
import { Card, CardContent } from "@/components/ui/card";
import { Badge, type BadgeProps } from "@/components/ui/badge";
import TaskDetailModal from "@/features/tasks/TaskDetailModal";

const COLUMNS: { status: TaskStatus; label: string }[] = [
  { status: "backlog", label: "Backlog" },
  { status: "todo", label: "Todo" },
  { status: "doing", label: "Doing" },
  { status: "review", label: "Review" },
  { status: "testing", label: "Testing" },
  { status: "done", label: "Done" },
  { status: "blocked", label: "Blocked" },
];

const PRIORITY_VARIANT: Record<TaskPriority, BadgeProps["variant"]> = {
  critical: "destructive",
  high: "primary",
  medium: "default",
  low: "outline",
};

export default function KanbanBoard() {
  const { projectId } = useParams<{ projectId: string }>();
  const queryClient = useQueryClient();
  const [openTaskId, setOpenTaskId] = useState<string | null>(null);

  const { data: tasks } = useQuery({
    queryKey: ["tasks", projectId],
    queryFn: () => listTasks(projectId!),
    enabled: !!projectId,
  });

  useTaskSocket(projectId);

  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: TaskStatus }) => updateTaskStatus(id, status),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["tasks", projectId] }),
  });

  function handleDragEnd(event: DragEndEvent) {
    const taskId = event.active.id as string;
    const newStatus = event.over?.id as TaskStatus | undefined;
    if (!newStatus) return;

    const task = tasks?.items.find((t) => t.id === taskId);
    if (task && task.status !== newStatus) {
      statusMutation.mutate({ id: taskId, status: newStatus });
    }
  }

  return (
    <div className="p-6">
      <DndContext onDragEnd={handleDragEnd}>
        <div className="flex gap-4 overflow-x-auto pb-4">
          {COLUMNS.map((col) => (
            <BoardColumn
              key={col.status}
              status={col.status}
              label={col.label}
              tasks={tasks?.items.filter((t) => t.status === col.status) ?? []}
              onOpenTask={setOpenTaskId}
            />
          ))}
        </div>
      </DndContext>

      {openTaskId && <TaskDetailModal taskId={openTaskId} onClose={() => setOpenTaskId(null)} />}
    </div>
  );
}

function BoardColumn({
  status,
  label,
  tasks,
  onOpenTask,
}: {
  status: TaskStatus;
  label: string;
  tasks: Task[];
  onOpenTask: (id: string) => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: status });

  return (
    <div
      ref={setNodeRef}
      className={`flex w-72 shrink-0 flex-col gap-2 rounded-xl p-2 transition-colors ${isOver ? "bg-primary/10" : ""}`}
    >
      <div className="flex items-center justify-between px-1">
        <h2 className="text-sm font-semibold text-muted-foreground">{label}</h2>
        <span className="text-xs text-muted-foreground">{tasks.length}</span>
      </div>
      {tasks.map((task) => (
        <TaskCard key={task.id} task={task} onOpen={() => onOpenTask(task.id)} />
      ))}
    </div>
  );
}

function TaskCard({ task, onOpen }: { task: Task; onOpen: () => void }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: task.id });

  const style = transform ? { transform: CSS.Translate.toString(transform), zIndex: 10 } : undefined;

  return (
    <Card
      ref={setNodeRef}
      style={style}
      {...listeners}
      {...attributes}
      className={`cursor-grab touch-none active:cursor-grabbing ${isDragging ? "opacity-50" : ""}`}
    >
      <CardContent className="p-3">
        <button className="text-left" onClick={onOpen}>
          <p className="text-sm font-medium hover:underline">{task.title}</p>
        </button>
        <div className="mt-2 flex flex-wrap items-center gap-1">
          <Badge variant={PRIORITY_VARIANT[task.priority]}>{task.priority}</Badge>
          {task.tags.map((tag) => (
            <Badge key={tag} variant="default">
              {tag}
            </Badge>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit a project's Board tab: confirm no duplicate header remains, drag-and-drop between columns still updates status (watch the Network tab for the `PATCH`/`PUT` call), and priority/tags render as Badges.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/tasks/KanbanBoard.tsx
git commit -m "$(cat <<'EOF'
Migrate KanbanBoard onto ProjectLayout and design-system primitives

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 11: Migrate CalendarPage

**Files:**
- Modify: `src/features/tasks/CalendarPage.tsx` (full rewrite, current file is 139 lines)

**Interfaces:**
- Consumes: `Select` (Task 2). Drops its own "← List view" header — owned by `ProjectLayout`. FullCalendar setup/drag-resize handlers are untouched.

- [ ] **Step 1: Replace the full contents of `src/features/tasks/CalendarPage.tsx`**

```tsx
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import FullCalendar from "@fullcalendar/react";
import dayGridPlugin from "@fullcalendar/daygrid";
import timeGridPlugin from "@fullcalendar/timegrid";
import interactionPlugin from "@fullcalendar/interaction";
import type { EventDropArg } from "@fullcalendar/core";
import type { EventResizeDoneArg } from "@fullcalendar/interaction";
import { listTasks, updateTaskDates } from "@/features/tasks/api";
import type { Task, TaskPriority, TaskStatus } from "@/features/tasks/types";
import { Select } from "@/components/ui/select";

const PRIORITY_BG: Record<TaskPriority, string> = {
  critical: "hsl(var(--destructive))",
  high: "hsl(var(--primary))",
  medium: "hsl(215 16% 47%)",
  low: "hsl(215 16% 65%)",
};

const STATUSES: TaskStatus[] = ["backlog", "todo", "doing", "review", "testing", "done", "blocked"];

function formatLocalDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function taskToEvent(t: Task) {
  const start = t.start_date ?? t.due_date;
  if (!start) return null;

  let end: string | undefined;
  if (t.due_date) {
    const endDate = new Date(`${t.due_date}T00:00:00`);
    endDate.setDate(endDate.getDate() + 1);
    end = formatLocalDate(endDate);
  }

  return {
    id: t.id,
    title: t.title,
    start,
    end,
    allDay: true,
    backgroundColor: PRIORITY_BG[t.priority],
    borderColor: PRIORITY_BG[t.priority],
  };
}

function eventToDates(start: Date, end: Date | null): { start_date?: string; due_date: string } {
  const startStr = formatLocalDate(start);
  let due = start;
  if (end) {
    due = new Date(end);
    due.setDate(due.getDate() - 1);
  }
  const dueStr = formatLocalDate(due);
  return startStr === dueStr ? { due_date: dueStr } : { start_date: startStr, due_date: dueStr };
}

export default function CalendarPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<TaskStatus | "">("");

  const { data: tasks } = useQuery({
    queryKey: ["tasks", projectId, { status }],
    queryFn: () => listTasks(projectId!, status ? { status } : undefined),
    enabled: !!projectId,
  });

  const moveMutation = useMutation({
    mutationFn: ({ id, dates }: { id: string; dates: { start_date?: string; due_date: string } }) =>
      updateTaskDates(id, dates),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["tasks", projectId] }),
  });

  function handleDrop(info: EventDropArg) {
    moveMutation.mutate({ id: info.event.id, dates: eventToDates(info.event.start!, info.event.end) });
  }

  function handleResize(info: EventResizeDoneArg) {
    moveMutation.mutate({ id: info.event.id, dates: eventToDates(info.event.start!, info.event.end) });
  }

  const events = (tasks?.items ?? []).map(taskToEvent).filter((e) => e !== null);

  return (
    <div className="p-6">
      <div className="mb-6 flex justify-end">
        <Select value={status} onChange={(e) => setStatus(e.target.value as TaskStatus | "")} className="w-48">
          <option value="">All statuses</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </Select>
      </div>

      <div className="glass rounded-xl p-4">
        <FullCalendar
          plugins={[dayGridPlugin, timeGridPlugin, interactionPlugin]}
          initialView="dayGridMonth"
          headerToolbar={{
            left: "prev,next today",
            center: "title",
            right: "dayGridMonth,timeGridWeek,timeGridDay",
          }}
          events={events}
          editable
          eventResizableFromStart
          eventDrop={handleDrop}
          eventResize={handleResize}
          height="auto"
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit a project's Calendar tab: confirm no duplicate header, the status filter still filters events, and dragging/resizing an event still persists (check Network tab).

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/tasks/CalendarPage.tsx
git commit -m "$(cat <<'EOF'
Migrate CalendarPage onto ProjectLayout and design-system primitives

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 12: Migrate GanttPage

**Files:**
- Modify: `src/features/tasks/GanttPage.tsx` (full rewrite, current file is 304 lines)

**Interfaces:**
- Consumes: `Tabs`/`TabButton` (Task 3, replaces the ad-hoc zoom-level button group), `Select` (Task 2), `EmptyState` (Task 2). Drops its own "← List view" header — owned by `ProjectLayout`. The timeline/SVG-dependency-arrow rendering math is untouched.

- [ ] **Step 1: Replace the full contents of `src/features/tasks/GanttPage.tsx`**

```tsx
import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { addDependency, getGanttView, removeDependency } from "@/features/tasks/api";
import type { Task, TaskPriority } from "@/features/tasks/types";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { Tabs, TabButton } from "@/components/ui/tabs";
import { EmptyState } from "@/components/ui/empty-state";

type Zoom = "day" | "week" | "month";
const DAY_WIDTH: Record<Zoom, number> = { day: 36, week: 14, month: 5 };
const ROW_HEIGHT = 36;
const LABEL_WIDTH = 240;

const PRIORITY_BG: Record<TaskPriority, string> = {
  critical: "hsl(var(--destructive))",
  high: "hsl(var(--primary))",
  medium: "hsl(215 16% 47%)",
  low: "hsl(215 16% 65%)",
};

function parseLocalDate(s: string): Date {
  return new Date(`${s}T00:00:00`);
}

function diffDays(a: Date, b: Date): number {
  return Math.round((b.getTime() - a.getTime()) / 86400000);
}

function addDays(d: Date, n: number): Date {
  const copy = new Date(d);
  copy.setDate(copy.getDate() + n);
  return copy;
}

interface Row {
  task: Task;
  start: Date;
  end: Date;
  durationDays: number;
  isMilestone: boolean;
}

export default function GanttPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const queryClient = useQueryClient();
  const [zoom, setZoom] = useState<Zoom>("day");
  const [successorId, setSuccessorId] = useState("");
  const [predecessorId, setPredecessorId] = useState("");

  const { data: gantt } = useQuery({
    queryKey: ["gantt", projectId],
    queryFn: () => getGanttView(projectId!),
    enabled: !!projectId,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["gantt", projectId] });

  const addDepMutation = useMutation({
    mutationFn: () => addDependency(successorId, predecessorId),
    onSuccess: () => {
      invalidate();
      setSuccessorId("");
      setPredecessorId("");
    },
  });

  const removeDepMutation = useMutation({
    mutationFn: ({ taskId, dependsOnId }: { taskId: string; dependsOnId: string }) => removeDependency(taskId, dependsOnId),
    onSuccess: () => invalidate(),
  });

  const dayWidth = DAY_WIDTH[zoom];
  const criticalSet = useMemo(() => new Set(gantt?.critical_path ?? []), [gantt]);

  const { rows, unscheduled, minDate, totalDays } = useMemo(() => {
    const tasks = gantt?.tasks ?? [];
    const scheduled: Row[] = [];
    const unscheduled: Task[] = [];

    for (const t of tasks) {
      if (!t.start_date && !t.due_date) {
        unscheduled.push(t);
        continue;
      }
      const start = parseLocalDate(t.start_date ?? t.due_date!);
      const end = parseLocalDate(t.due_date ?? t.start_date!);
      scheduled.push({ task: t, start, end, durationDays: diffDays(start, end), isMilestone: diffDays(start, end) === 0 });
    }
    scheduled.sort((a, b) => a.start.getTime() - b.start.getTime());

    if (scheduled.length === 0) {
      return { rows: [] as Row[], unscheduled, minDate: new Date(), totalDays: 0 };
    }

    let minDate = scheduled[0].start;
    let maxDate = scheduled[0].end;
    for (const r of scheduled) {
      if (r.start < minDate) minDate = r.start;
      if (r.end > maxDate) maxDate = r.end;
    }
    minDate = addDays(minDate, -1);
    maxDate = addDays(maxDate, 1);

    return { rows: scheduled, unscheduled, minDate, totalDays: diffDays(minDate, maxDate) + 1 };
  }, [gantt]);

  const rowIndex = useMemo(() => {
    const map = new Map<string, number>();
    rows.forEach((r, i) => map.set(r.task.id, i));
    return map;
  }, [rows]);

  const timelineWidth = totalDays * dayWidth;

  const ticks = useMemo(() => {
    if (totalDays === 0) return [];
    const step = zoom === "day" ? 1 : zoom === "week" ? 7 : 30;
    const result: { label: string; offset: number }[] = [];
    for (let d = 0; d < totalDays; d += step) {
      const date = addDays(minDate, d);
      result.push({
        label: date.toLocaleDateString(undefined, zoom === "month" ? { month: "short", year: "2-digit" } : { month: "short", day: "numeric" }),
        offset: d * dayWidth,
      });
    }
    return result;
  }, [minDate, totalDays, zoom, dayWidth]);

  return (
    <div className="p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          {gantt && gantt.project_days > 0 && (
            <p className="text-sm text-muted-foreground">Critical path: {gantt.project_days} day{gantt.project_days === 1 ? "" : "s"}</p>
          )}
        </div>
        <Tabs>
          {(["day", "week", "month"] as Zoom[]).map((z) => (
            <TabButton key={z} active={zoom === z} onClick={() => setZoom(z)}>
              {z[0].toUpperCase() + z.slice(1)}
            </TabButton>
          ))}
        </Tabs>
      </div>

      {rows.length === 0 ? (
        <EmptyState title="No tasks with dates yet" description="Add a due date on the List view to see them here." />
      ) : (
        <div className="glass overflow-auto rounded-xl">
          <div className="flex" style={{ width: LABEL_WIDTH + timelineWidth }}>
            <div className="sticky left-0 z-10 shrink-0 border-r border-border bg-card" style={{ width: LABEL_WIDTH }}>
              <div className="flex items-center border-b border-border px-3 text-xs font-medium text-muted-foreground" style={{ height: ROW_HEIGHT }}>
                Task
              </div>
              {rows.map((r) => (
                <div key={r.task.id} className="flex items-center truncate border-b border-border px-3 text-sm" style={{ height: ROW_HEIGHT }}>
                  {r.task.title}
                </div>
              ))}
              {unscheduled.map((t) => (
                <div key={t.id} className="flex items-center truncate border-b border-border px-3 text-sm text-muted-foreground" style={{ height: ROW_HEIGHT }}>
                  {t.title} <span className="ml-1 text-xs">(no dates)</span>
                </div>
              ))}
            </div>

            <div className="relative" style={{ width: timelineWidth }}>
              <div className="relative border-b border-border" style={{ height: ROW_HEIGHT }}>
                {ticks.map((tick) => (
                  <div
                    key={tick.offset}
                    className="absolute top-0 h-full border-l border-border px-1 text-xs text-muted-foreground"
                    style={{ left: tick.offset }}
                  >
                    {tick.label}
                  </div>
                ))}
              </div>

              <svg className="pointer-events-none absolute left-0 top-[36px]" width={timelineWidth} height={rows.length * ROW_HEIGHT}>
                {(gantt?.dependencies ?? []).map((dep, i) => {
                  const fromIdx = rowIndex.get(dep.depends_on_task_id);
                  const toIdx = rowIndex.get(dep.task_id);
                  if (fromIdx === undefined || toIdx === undefined) return null;
                  const fromRow = rows[fromIdx];
                  const fromX = (diffDays(minDate, fromRow.end) || 0) * dayWidth;
                  const fromY = fromIdx * ROW_HEIGHT + ROW_HEIGHT / 2;
                  const toRow = rows[toIdx];
                  const toX = diffDays(minDate, toRow.start) * dayWidth;
                  const toY = toIdx * ROW_HEIGHT + ROW_HEIGHT / 2;
                  const midX = fromX + Math.max(8, (toX - fromX) / 2);
                  return (
                    <path
                      key={i}
                      d={`M ${fromX} ${fromY} L ${midX} ${fromY} L ${midX} ${toY} L ${toX} ${toY}`}
                      fill="none"
                      stroke="hsl(var(--muted-foreground))"
                      strokeWidth={1.5}
                      markerEnd="url(#arrow)"
                    />
                  );
                })}
                <defs>
                  <marker id="arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">
                    <path d="M0,0 L6,3 L0,6 Z" fill="hsl(var(--muted-foreground))" />
                  </marker>
                </defs>
              </svg>

              {rows.map((r, i) => {
                const critical = criticalSet.has(r.task.id);
                const color = PRIORITY_BG[r.task.priority];
                const top = i * ROW_HEIGHT + 6;
                if (r.isMilestone) {
                  const x = diffDays(minDate, r.start) * dayWidth;
                  return (
                    <div
                      key={r.task.id}
                      title={r.task.title}
                      className={`absolute rotate-45 ${critical ? "ring-2 ring-destructive" : ""}`}
                      style={{ left: x - 8, top, width: 16, height: 16, backgroundColor: color }}
                    />
                  );
                }
                const left = diffDays(minDate, r.start) * dayWidth;
                const width = Math.max(r.durationDays * dayWidth, 6);
                return (
                  <div
                    key={r.task.id}
                    title={r.task.title}
                    className={`absolute rounded-md ${critical ? "ring-2 ring-destructive" : ""}`}
                    style={{ left, top, width, height: ROW_HEIGHT - 12, backgroundColor: color }}
                  />
                );
              })}
            </div>
          </div>
        </div>
      )}

      <div className="glass mt-6 rounded-xl p-4">
        <h2 className="mb-3 text-sm font-semibold">Dependencies</h2>
        <form
          className="mb-4 flex flex-wrap items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (successorId && predecessorId && successorId !== predecessorId) addDepMutation.mutate();
          }}
        >
          <Select value={successorId} onChange={(e) => setSuccessorId(e.target.value)} className="h-9 w-48">
            <option value="">Task…</option>
            {gantt?.tasks.map((t) => (
              <option key={t.id} value={t.id}>{t.title}</option>
            ))}
          </Select>
          <span className="text-sm text-muted-foreground">depends on</span>
          <Select value={predecessorId} onChange={(e) => setPredecessorId(e.target.value)} className="h-9 w-48">
            <option value="">Task…</option>
            {gantt?.tasks.map((t) => (
              <option key={t.id} value={t.id}>{t.title}</option>
            ))}
          </Select>
          <Button type="submit" size="sm" disabled={addDepMutation.isPending || !successorId || !predecessorId}>
            Add dependency
          </Button>
          {addDepMutation.isError && <span className="text-sm text-destructive">Could not add — check for a cycle.</span>}
        </form>

        {(gantt?.dependencies ?? []).length === 0 ? (
          <p className="text-sm text-muted-foreground">No dependencies yet.</p>
        ) : (
          <ul className="flex flex-col gap-1">
            {gantt?.dependencies.map((dep, i) => {
              const successor = gantt.tasks.find((t) => t.id === dep.task_id);
              const predecessor = gantt.tasks.find((t) => t.id === dep.depends_on_task_id);
              return (
                <li key={i} className="flex items-center justify-between text-sm">
                  <span>
                    <strong>{successor?.title ?? "?"}</strong> depends on <strong>{predecessor?.title ?? "?"}</strong>
                  </span>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => removeDepMutation.mutate({ taskId: dep.task_id, dependsOnId: dep.depends_on_task_id })}
                  >
                    Remove
                  </Button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit a project's Gantt tab: confirm no duplicate header, the Day/Week/Month zoom control now renders via `Tabs`/`TabButton` (same look, same behavior), the timeline/critical-path/dependency-arrow rendering is pixel-identical to before, and adding/removing a dependency still works.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/tasks/GanttPage.tsx
git commit -m "$(cat <<'EOF'
Migrate GanttPage onto ProjectLayout and design-system primitives

Zoom control now uses the shared Tabs/TabButton primitive instead of
its own ad-hoc button group. Timeline rendering math is unchanged.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 13: Migrate ActionPlanPage

**Files:**
- Modify: `src/features/actionplan/ActionPlanPage.tsx` (full rewrite, current file is 229 lines)

**Interfaces:**
- Consumes: `Badge`/`BadgeProps`, `Input`, `Select`, `EmptyState` (Task 2). Drops its own "← List view" header — owned by `ProjectLayout`.

- [ ] **Step 1: Replace the full contents of `src/features/actionplan/ActionPlanPage.tsx`**

```tsx
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { assignTaskToMilestone, createGoal, createMilestone, getActionPlan } from "@/features/actionplan/api";
import type { GoalNode, MilestoneNode } from "@/features/actionplan/types";
import { listTasks } from "@/features/tasks/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Badge, type BadgeProps } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";

const STATUS_LABEL: Record<string, string> = {
  not_started: "Not started",
  in_progress: "In progress",
  completed: "Completed",
};

const STATUS_VARIANT: Record<string, BadgeProps["variant"]> = {
  not_started: "default",
  in_progress: "primary",
  completed: "outline",
};

export default function ActionPlanPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const queryClient = useQueryClient();

  const [showGoalForm, setShowGoalForm] = useState(false);
  const [goalName, setGoalName] = useState("");
  const [milestoneFormGoalId, setMilestoneFormGoalId] = useState<string | null>(null);
  const [milestoneName, setMilestoneName] = useState("");
  const [assignFormMilestoneId, setAssignFormMilestoneId] = useState<string | null>(null);
  const [assignTaskId, setAssignTaskId] = useState("");

  const { data: plan, isLoading } = useQuery({
    queryKey: ["actionplan", projectId],
    queryFn: () => getActionPlan(projectId!),
    enabled: !!projectId,
  });

  const { data: allTasks } = useQuery({
    queryKey: ["tasks", projectId],
    queryFn: () => listTasks(projectId!),
    enabled: !!projectId,
  });
  const unplannedTasks = (allTasks?.items ?? []).filter((t) => !t.milestone_id && !t.parent_task_id);

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["actionplan", projectId] });
    queryClient.invalidateQueries({ queryKey: ["tasks", projectId] });
  };

  const goalMutation = useMutation({
    mutationFn: () => createGoal({ project_id: projectId!, name: goalName }),
    onSuccess: () => {
      invalidate();
      setGoalName("");
      setShowGoalForm(false);
    },
  });

  const milestoneMutation = useMutation({
    mutationFn: (goalId: string) => createMilestone({ goal_id: goalId, name: milestoneName }),
    onSuccess: () => {
      invalidate();
      setMilestoneName("");
      setMilestoneFormGoalId(null);
    },
  });

  const assignMutation = useMutation({
    mutationFn: ({ taskId, milestoneId }: { taskId: string; milestoneId: string }) => assignTaskToMilestone(taskId, milestoneId),
    onSuccess: () => {
      invalidate();
      setAssignTaskId("");
      setAssignFormMilestoneId(null);
    },
  });

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-end">
        <Button onClick={() => setShowGoalForm((v) => !v)}>{showGoalForm ? "Cancel" : "New Goal"}</Button>
      </div>

      {showGoalForm && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create goal</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="flex gap-3"
              onSubmit={(e) => {
                e.preventDefault();
                goalMutation.mutate();
              }}
            >
              <Input required placeholder="Goal name" value={goalName} onChange={(e) => setGoalName(e.target.value)} className="flex-1" />
              <Button type="submit" disabled={goalMutation.isPending}>
                {goalMutation.isPending ? "Creating…" : "Create"}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {!isLoading && plan?.goals.length === 0 && (
        <EmptyState title="No goals yet" description="Create one to start structuring this project's roadmap." />
      )}

      <div className="flex flex-col gap-4">
        {plan?.goals.map((goal: GoalNode) => (
          <Card key={goal.id}>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>{goal.name}</CardTitle>
                <Badge variant={STATUS_VARIANT[goal.status]} className="mt-1">
                  {STATUS_LABEL[goal.status]}
                </Badge>
              </div>
              <Button size="sm" variant="outline" onClick={() => setMilestoneFormGoalId(milestoneFormGoalId === goal.id ? null : goal.id)}>
                {milestoneFormGoalId === goal.id ? "Cancel" : "+ Milestone"}
              </Button>
            </CardHeader>
            <CardContent>
              {milestoneFormGoalId === goal.id && (
                <form
                  className="mb-4 flex gap-2"
                  onSubmit={(e) => {
                    e.preventDefault();
                    milestoneMutation.mutate(goal.id);
                  }}
                >
                  <Input
                    required
                    placeholder="Milestone name"
                    value={milestoneName}
                    onChange={(e) => setMilestoneName(e.target.value)}
                    className="h-9 flex-1"
                  />
                  <Button type="submit" size="sm" disabled={milestoneMutation.isPending}>
                    Add
                  </Button>
                </form>
              )}

              {goal.milestones.length === 0 ? (
                <p className="text-sm text-muted-foreground">No milestones yet.</p>
              ) : (
                <div className="flex flex-col gap-3">
                  {goal.milestones.map((milestone: MilestoneNode) => (
                    <div key={milestone.id} className="rounded-lg border border-border p-3">
                      <div className="mb-2 flex items-center justify-between">
                        <div>
                          <p className="text-sm font-medium">{milestone.name}</p>
                          <Badge variant={STATUS_VARIANT[milestone.status]} className="mt-1">
                            {STATUS_LABEL[milestone.status]}
                          </Badge>
                        </div>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => setAssignFormMilestoneId(assignFormMilestoneId === milestone.id ? null : milestone.id)}
                        >
                          {assignFormMilestoneId === milestone.id ? "Cancel" : "+ Task"}
                        </Button>
                      </div>

                      {assignFormMilestoneId === milestone.id && (
                        <form
                          className="mb-3 flex gap-2"
                          onSubmit={(e) => {
                            e.preventDefault();
                            if (assignTaskId) assignMutation.mutate({ taskId: assignTaskId, milestoneId: milestone.id });
                          }}
                        >
                          <Select value={assignTaskId} onChange={(e) => setAssignTaskId(e.target.value)} className="h-9 flex-1">
                            <option value="">Select an unplanned task…</option>
                            {unplannedTasks.map((t) => (
                              <option key={t.id} value={t.id}>{t.title}</option>
                            ))}
                          </Select>
                          <Button type="submit" size="sm" disabled={!assignTaskId || assignMutation.isPending}>
                            Assign
                          </Button>
                        </form>
                      )}

                      {milestone.tasks.length === 0 ? (
                        <p className="text-xs text-muted-foreground">No tasks assigned.</p>
                      ) : (
                        <ul className="flex flex-col gap-1">
                          {milestone.tasks.map((task) => (
                            <li key={task.id} className="text-sm">
                              <span>{task.title}</span>
                              {task.subtasks.length > 0 && (
                                <ul className="ml-4 mt-1 flex flex-col gap-0.5 border-l border-border pl-3">
                                  {task.subtasks.map((sub) => (
                                    <li key={sub.id} className="text-xs text-muted-foreground">{sub.title}</li>
                                  ))}
                                </ul>
                              )}
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit a project's Action Plan tab: confirm no duplicate header, goal/milestone status now renders as Badges, and creating a goal/milestone/task assignment still works.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/actionplan/ActionPlanPage.tsx
git commit -m "$(cat <<'EOF'
Migrate ActionPlanPage onto ProjectLayout and design-system primitives

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 14: Migrate ReportsPage

**Files:**
- Modify: `src/features/reports/ReportsPage.tsx` (full rewrite, current file is 160 lines)

**Interfaces:**
- Consumes: `Tabs`/`TabButton` (Task 3, replaces the ad-hoc Weekly/Monthly button group). Drops its own "← List view" header — owned by `ProjectLayout`, which already carries `print:hidden` (Task 5) so the print stylesheet still hides navigation chrome above this page. Unlike the other 5 project-scoped pages (Tasks 9–13), this one **keeps** its own `getProject` query — it needs `project?.name` for the print-only `<h2 className="print:block">` title, which must render even though `ProjectLayout`'s (non-print) header is hidden during print.

- [ ] **Step 1: Replace the full contents of `src/features/reports/ReportsPage.tsx`**

```tsx
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { getProject, getProjectMembers } from "@/features/projects/api";
import { downloadCsv, getPeriodReport } from "@/features/reports/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Tabs, TabButton } from "@/components/ui/tabs";

function StatTile({ label, value }: { label: string; value: string | number }) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="mt-1 text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}

export default function ReportsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [period, setPeriod] = useState<"week" | "month">("week");

  const { data: project } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => getProject(projectId!),
    enabled: !!projectId,
  });

  const { data: report, isLoading } = useQuery({
    queryKey: ["report", projectId, period],
    queryFn: () => getPeriodReport(projectId!, period),
    enabled: !!projectId,
  });

  const { data: members } = useQuery({
    queryKey: ["project-members", projectId],
    queryFn: () => getProjectMembers(projectId!),
    enabled: !!projectId,
  });
  const nameByUserId = new Map((members ?? []).map((m) => [m.user_id, m.name]));

  return (
    <div className="p-6 print:p-0">
      <div className="mb-6 flex flex-wrap items-center justify-end gap-3 print:hidden">
        <Tabs>
          <TabButton active={period === "week"} onClick={() => setPeriod("week")}>
            Weekly
          </TabButton>
          <TabButton active={period === "month"} onClick={() => setPeriod("month")}>
            Monthly
          </TabButton>
        </Tabs>
        <Button variant="outline" onClick={() => downloadCsv(projectId!)}>Export CSV</Button>
        <Button variant="outline" onClick={() => window.print()}>Save as PDF</Button>
      </div>

      {isLoading && <p className="text-muted-foreground">Loading report…</p>}

      {report && (
        <div className="flex flex-col gap-6">
          <h2 className="hidden text-xl font-semibold print:block">
            {project?.name} — {period === "week" ? "Weekly" : "Monthly"} Report ({report.from} to {report.to})
          </h2>

          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
            <StatTile label="Total Tasks" value={report.summary.total_tasks} />
            <StatTile label="Created" value={report.summary.created_in_period} />
            <StatTile label="Completed" value={report.summary.completed_in_period} />
            <StatTile label="Completion Rate" value={`${Math.round(report.summary.completion_rate * 100)}%`} />
            <StatTile label="Overdue" value={report.summary.overdue_count} />
            <StatTile label="Near Due" value={report.summary.near_due_count} />
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Productivity</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={report.productivity}>
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                    <XAxis dataKey="date" tick={{ fontSize: 12 }} />
                    <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                    <Tooltip />
                    <Bar dataKey="completed_count" name="Completed" fill="hsl(var(--primary))" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Performance by Assignee</CardTitle>
            </CardHeader>
            <CardContent>
              {report.performance.length === 0 ? (
                <p className="text-sm text-muted-foreground">No tasks completed in this period yet.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border text-left text-xs text-muted-foreground">
                        <th className="py-2 pr-4">Assignee</th>
                        <th className="py-2 pr-4">Completed</th>
                        <th className="py-2 pr-4">On Time</th>
                        <th className="py-2 pr-4">Late</th>
                        <th className="py-2 pr-4">On-Time Rate</th>
                      </tr>
                    </thead>
                    <tbody>
                      {report.performance.map((p) => (
                        <tr key={p.assignee_id} className="border-b border-border last:border-0">
                          <td className="py-2 pr-4">{nameByUserId.get(p.assignee_id) ?? p.assignee_id}</td>
                          <td className="py-2 pr-4">{p.completed_count}</td>
                          <td className="py-2 pr-4">{p.on_time_count}</td>
                          <td className="py-2 pr-4">{p.late_count}</td>
                          <td className="py-2 pr-4">{Math.round(p.on_time_rate * 100)}%</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Late Tasks ({report.late_tasks.length})</CardTitle>
            </CardHeader>
            <CardContent>
              {report.late_tasks.length === 0 ? (
                <p className="text-sm text-muted-foreground">Nothing overdue. 🎉</p>
              ) : (
                <ul className="flex flex-col gap-1">
                  {report.late_tasks.map((t) => (
                    <li key={t.id} className="flex items-center justify-between text-sm">
                      <span>{t.title}</span>
                      <span className="text-destructive">Due {t.due_date}</span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit a project's Reports tab: confirm no duplicate header, Weekly/Monthly now renders via `Tabs`/`TabButton`, then use the browser's print preview (`Ctrl/Cmd+P` or the "Save as PDF" button) and confirm the sidebar, top bar, and ProjectLayout tab bar are all absent from the print preview — only the report content and its print-only title show.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/reports/ReportsPage.tsx
git commit -m "$(cat <<'EOF'
Migrate ReportsPage onto ProjectLayout and design-system primitives

Period toggle now uses the shared Tabs/TabButton primitive.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 15: Migrate TeamPage

**Files:**
- Modify: `src/features/team/TeamPage.tsx` (full rewrite, current file is 153 lines)

**Interfaces:**
- Consumes: `Avatar`, `Input`, `Select`, `Skeleton` (Task 2). Drops the "← Dashboard" back-link (redundant now that the sidebar is always visible) but keeps its own `<h1>Team</h1>` — `AppShell` doesn't render per-page titles for top-level routes.

- [ ] **Step 1: Replace the full contents of `src/features/team/TeamPage.tsx`**

```tsx
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/auth-store";
import { createDepartment, getDirectory, updateUserDepartment, updateUserRole } from "@/features/team/api";
import type { DirectoryEntry, OrgRole } from "@/features/team/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Avatar } from "@/components/ui/avatar";
import { Skeleton } from "@/components/ui/skeleton";

const ROLES: OrgRole[] = ["super_admin", "admin", "manager", "leader", "employee"];

export default function TeamPage() {
  const currentUser = useAuthStore((s) => s.user);
  const isAdmin = currentUser?.role === "super_admin" || currentUser?.role === "admin";
  const queryClient = useQueryClient();
  const [showDeptForm, setShowDeptForm] = useState(false);
  const [deptName, setDeptName] = useState("");

  const { data: directory, isLoading } = useQuery({ queryKey: ["team"], queryFn: getDirectory });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["team"] });

  const deptMutation = useMutation({
    mutationFn: () => createDepartment(deptName),
    onSuccess: () => {
      invalidate();
      setDeptName("");
      setShowDeptForm(false);
    },
  });

  const roleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: OrgRole }) => updateUserRole(userId, role),
    onSuccess: invalidate,
  });

  const deptAssignMutation = useMutation({
    mutationFn: ({ userId, departmentId }: { userId: string; departmentId: string | null }) => updateUserDepartment(userId, departmentId),
    onSuccess: invalidate,
  });

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Team</h1>
        {isAdmin && (
          <Button variant="outline" onClick={() => setShowDeptForm((v) => !v)}>
            {showDeptForm ? "Cancel" : "New Department"}
          </Button>
        )}
      </div>

      {showDeptForm && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create department</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="flex gap-3"
              onSubmit={(e) => {
                e.preventDefault();
                deptMutation.mutate();
              }}
            >
              <Input required placeholder="Department name" value={deptName} onChange={(e) => setDeptName(e.target.value)} className="flex-1" />
              <Button type="submit" disabled={deptMutation.isPending}>
                {deptMutation.isPending ? "Creating…" : "Create"}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {isLoading && (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      )}

      {directory && (
        <Card>
          <CardContent className="overflow-x-auto p-0">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-left text-xs text-muted-foreground">
                  <th className="p-3">Name</th>
                  <th className="p-3">Email</th>
                  <th className="p-3">Role</th>
                  <th className="p-3">Department</th>
                  <th className="p-3">Status</th>
                  <th className="p-3">Workload</th>
                </tr>
              </thead>
              <tbody>
                {directory.members.map((m: DirectoryEntry) => (
                  <tr key={m.user_id} className="border-b border-border last:border-0">
                    <td className="p-3">
                      <span className="flex items-center gap-2 font-medium">
                        <Avatar name={m.name} />
                        {m.name}
                      </span>
                    </td>
                    <td className="p-3 text-muted-foreground">{m.email}</td>
                    <td className="p-3">
                      {isAdmin ? (
                        <Select
                          value={m.role}
                          onChange={(e) => roleMutation.mutate({ userId: m.user_id, role: e.target.value as OrgRole })}
                          className="h-8 w-36 text-xs"
                        >
                          {ROLES.map((r) => (
                            <option key={r} value={r}>{r}</option>
                          ))}
                        </Select>
                      ) : (
                        <span className="text-xs">{m.role}</span>
                      )}
                    </td>
                    <td className="p-3">
                      {isAdmin ? (
                        <Select
                          value={m.department_id ?? ""}
                          onChange={(e) => deptAssignMutation.mutate({ userId: m.user_id, departmentId: e.target.value || null })}
                          className="h-8 w-36 text-xs"
                        >
                          <option value="">—</option>
                          {directory.departments.map((d) => (
                            <option key={d.id} value={d.id}>{d.name}</option>
                          ))}
                        </Select>
                      ) : (
                        <span className="text-xs text-muted-foreground">{m.department_name ?? "—"}</span>
                      )}
                    </td>
                    <td className="p-3">
                      <span className="inline-flex items-center gap-1.5 text-xs">
                        <span className={`h-2 w-2 rounded-full ${m.online ? "bg-primary" : "bg-muted-foreground"}`} />
                        {m.online ? "Online" : "Offline"}
                      </span>
                    </td>
                    <td className="p-3">{m.workload} active</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit `/team`: confirm no "← Dashboard" link remains, each member row shows an Avatar, and (as an admin) role/department dropdowns still update on change.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/team/TeamPage.tsx
git commit -m "$(cat <<'EOF'
Migrate TeamPage onto the new shell and design-system primitives

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 16: Migrate GamificationPage

**Files:**
- Modify: `src/features/gamification/GamificationPage.tsx` (full rewrite, current file is 183 lines)

**Interfaces:**
- Consumes: `Tabs`/`TabButton` (Task 3, replaces the ad-hoc People/Departments button group), `Avatar` (Task 2, leaderboard rows). Drops the "← Dashboard" back-link but keeps its own `<h1>Gamification</h1>`.

- [ ] **Step 1: Replace the full contents of `src/features/gamification/GamificationPage.tsx`**

```tsx
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getLeaderboard, getProfile } from "@/features/gamification/api";
import type { BadgeCode, MissionType } from "@/features/gamification/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Avatar } from "@/components/ui/avatar";
import { Tabs, TabButton } from "@/components/ui/tabs";

const BADGE_INFO: Record<BadgeCode, { label: string; emoji: string }> = {
  early_bird: { label: "Early Bird", emoji: "🐦" },
  fast_worker: { label: "Fast Worker", emoji: "⚡" },
  hundred_tasks: { label: "100 Tasks", emoji: "💯" },
  perfect_week: { label: "Perfect Week", emoji: "✨" },
  seven_day_streak: { label: "7 Day Streak", emoji: "🔥" },
  legend: { label: "Legend", emoji: "👑" },
};

const LEVEL_EMOJI: Record<string, string> = {
  Novice: "🌱",
  Adept: "🌿",
  Master: "🌳",
  Legend: "🏆",
};

const MISSION_LABEL: Record<MissionType, string> = {
  daily: "Daily Mission",
  weekly: "Weekly Mission",
  monthly: "Monthly Mission",
};

export default function GamificationPage() {
  const [leaderboardTab, setLeaderboardTab] = useState<"individual" | "department">("individual");

  const { data: profile, isLoading } = useQuery({ queryKey: ["gamification-profile"], queryFn: getProfile });
  const { data: leaderboard } = useQuery({ queryKey: ["gamification-leaderboard"], queryFn: getLeaderboard });

  return (
    <div className="p-6">
      <h1 className="mb-6 text-2xl font-semibold">Gamification</h1>

      {isLoading && <p className="text-muted-foreground">Loading profile…</p>}

      {profile && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="flex flex-col gap-6 lg:col-span-2">
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center gap-4">
                  <div className="flex h-16 w-16 items-center justify-center rounded-full bg-primary/10 text-3xl">
                    {LEVEL_EMOJI[profile.character.level_title] ?? "🌱"}
                  </div>
                  <div className="flex-1">
                    <p className="text-lg font-semibold">
                      Level {profile.character.level} — {profile.character.level_title}
                    </p>
                    <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full bg-primary transition-all"
                        style={{ width: `${(profile.character.exp_into_level / profile.character.exp_per_level) * 100}%` }}
                      />
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {profile.character.exp_into_level} / {profile.character.exp_per_level} EXP to next level · {profile.character.exp} total EXP
                    </p>
                  </div>
                </div>
                <div className="mt-4 grid grid-cols-3 gap-3 text-center">
                  <div>
                    <p className="text-xl font-semibold">{profile.character.total_completed}</p>
                    <p className="text-xs text-muted-foreground">Tasks completed</p>
                  </div>
                  <div>
                    <p className="text-xl font-semibold">🔥 {profile.character.current_streak}</p>
                    <p className="text-xs text-muted-foreground">Current streak</p>
                  </div>
                  <div>
                    <p className="text-xl font-semibold">{profile.character.longest_streak}</p>
                    <p className="text-xs text-muted-foreground">Longest streak</p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Badges</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-3 gap-3 sm:grid-cols-6">
                  {(Object.keys(BADGE_INFO) as BadgeCode[]).map((code) => {
                    const earned = profile.badges.some((b) => b.code === code);
                    const info = BADGE_INFO[code];
                    return (
                      <div
                        key={code}
                        title={info.label}
                        className={`flex flex-col items-center gap-1 rounded-lg border border-border p-3 text-center ${earned ? "" : "opacity-30 grayscale"}`}
                      >
                        <span className="text-2xl">{info.emoji}</span>
                        <span className="text-[10px] leading-tight">{info.label}</span>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Missions</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-4">
                {profile.missions.map((m) => (
                  <div key={m.type}>
                    <div className="mb-1 flex items-center justify-between text-sm">
                      <span>{MISSION_LABEL[m.type]}</span>
                      <span className="text-muted-foreground">
                        {m.count}/{m.threshold} {m.completed && "✓"} · +{m.reward} EXP
                      </span>
                    </div>
                    <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                      <div
                        className={`h-full transition-all ${m.completed ? "bg-primary" : "bg-primary/60"}`}
                        style={{ width: `${Math.min(100, (m.count / m.threshold) * 100)}%` }}
                      />
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>Leaderboard</CardTitle>
              <Tabs>
                <TabButton active={leaderboardTab === "individual"} onClick={() => setLeaderboardTab("individual")}>
                  People
                </TabButton>
                <TabButton active={leaderboardTab === "department"} onClick={() => setLeaderboardTab("department")}>
                  Departments
                </TabButton>
              </Tabs>
            </CardHeader>
            <CardContent>
              {leaderboardTab === "individual" ? (
                <ol className="flex flex-col gap-2">
                  {leaderboard?.individuals.map((entry, i) => (
                    <li key={entry.user_id} className="flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2">
                        <span className="w-5 text-muted-foreground">{i + 1}.</span>
                        <Avatar name={entry.name} className="h-6 w-6 text-[10px]" />
                        {entry.name}
                      </span>
                      <span className="text-muted-foreground">Lv.{entry.level} · {entry.exp} EXP</span>
                    </li>
                  ))}
                  {leaderboard?.individuals.length === 0 && <p className="text-sm text-muted-foreground">No activity yet.</p>}
                </ol>
              ) : (
                <ol className="flex flex-col gap-2">
                  {leaderboard?.departments.map((entry, i) => (
                    <li key={entry.department_id} className="flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2">
                        <span className="w-5 text-muted-foreground">{i + 1}.</span>
                        {entry.department_name}
                      </span>
                      <span className="text-muted-foreground">{entry.total_exp} EXP · {entry.member_count} members</span>
                    </li>
                  ))}
                  {leaderboard?.departments.length === 0 && <p className="text-sm text-muted-foreground">No departments with activity yet.</p>}
                </ol>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit `/gamification`: confirm no "← Dashboard" link remains, the People/Departments toggle now renders via `Tabs`/`TabButton`, and individual leaderboard rows show a small Avatar next to each name.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/gamification/GamificationPage.tsx
git commit -m "$(cat <<'EOF'
Migrate GamificationPage onto the new shell and design-system primitives

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 17: Migrate AIAssistantPage

**Files:**
- Modify: `src/features/ai/AIAssistantPage.tsx` (full rewrite, current file is 247 lines)

**Interfaces:**
- Consumes: `Input`, `Textarea`, `Select` (Task 2). Drops the "← Dashboard" back-link but keeps its own `<h1>AI Assistant</h1>` + subtitle. Drops the page-local `ErrorText` component and all 8 of its call sites — every one of these 8 `useMutation` calls now surfaces failures through the app-wide `MutationCache` error toast from Task 1, so the duplicate inline "Something went wrong" message is redundant (same reasoning as Task 8's attachment-upload error text removal).

- [ ] **Step 1: Replace the full contents of `src/features/ai/AIAssistantPage.tsx`**

```tsx
import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { listProjects } from "@/features/projects/api";
import {
  estimateDuration,
  generateMeetingSummary,
  generateTasks,
  getDailySummary,
  getProductivityAnalysis,
  getWeeklySummary,
  predictLateTasks,
  suggestAssignee,
  suggestPriority,
} from "@/features/ai/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Select } from "@/components/ui/select";

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">{children}</CardContent>
    </Card>
  );
}

export default function AIAssistantPage() {
  const { data: projects } = useQuery({ queryKey: ["projects-for-ai"], queryFn: listProjects });
  const [projectId, setProjectId] = useState("");

  const ProjectPicker = (
    <Select value={projectId} onChange={(e) => setProjectId(e.target.value)}>
      <option value="">Select a project…</option>
      {projects?.items.map((p) => (
        <option key={p.id} value={p.id}>
          {p.name}
        </option>
      ))}
    </Select>
  );

  // Generate Tasks
  const [genPrompt, setGenPrompt] = useState("");
  const genTasks = useMutation({ mutationFn: () => generateTasks({ project_id: projectId, prompt: genPrompt }) });

  // Daily summary
  const dailySummary = useMutation({ mutationFn: getDailySummary });

  // Weekly summary
  const weeklySummary = useMutation({ mutationFn: () => getWeeklySummary(projectId) });

  // Estimate duration
  const [estTitle, setEstTitle] = useState("");
  const [estDesc, setEstDesc] = useState("");
  const estimate = useMutation({ mutationFn: () => estimateDuration({ title: estTitle, description: estDesc }) });

  // Suggest priority
  const [prTitle, setPrTitle] = useState("");
  const [prDesc, setPrDesc] = useState("");
  const priority = useMutation({ mutationFn: () => suggestPriority({ title: prTitle, description: prDesc }) });

  // Suggest assignee
  const [asTitle, setAsTitle] = useState("");
  const [asDesc, setAsDesc] = useState("");
  const assignee = useMutation({ mutationFn: () => suggestAssignee({ project_id: projectId, title: asTitle, description: asDesc }) });

  // Predict late tasks
  const lateTasks = useMutation({ mutationFn: () => predictLateTasks(projectId) });

  // Productivity analysis
  const productivity = useMutation({ mutationFn: () => getProductivityAnalysis(projectId) });

  // Meeting summary
  const [notes, setNotes] = useState("");
  const meeting = useMutation({ mutationFn: () => generateMeetingSummary({ notes }) });

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">AI Assistant</h1>
        <p className="text-sm text-muted-foreground">Powered by Claude. Suggestions are a starting point — review before acting on them.</p>
      </div>

      <div className="mb-6 max-w-xs">
        <label className="mb-1 block text-xs font-medium text-muted-foreground">Project (for project-scoped tools)</label>
        {ProjectPicker}
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Section title="Generate Tasks">
          <Textarea
            rows={3}
            placeholder="Describe a goal or feature, e.g. 'Add password reset via email'"
            value={genPrompt}
            onChange={(e) => setGenPrompt(e.target.value)}
          />
          <Button size="sm" disabled={!projectId || !genPrompt || genTasks.isPending} onClick={() => genTasks.mutate()}>
            {genTasks.isPending ? "Generating…" : "Generate tasks"}
          </Button>
          {genTasks.data && (
            <ul className="flex flex-col gap-2">
              {genTasks.data.tasks.map((t, i) => (
                <li key={i} className="rounded-lg border border-border p-3 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="font-medium">{t.title}</span>
                    <span className="text-xs text-muted-foreground">{t.priority} · {t.estimate_hours}h</span>
                  </div>
                  <p className="mt-1 text-muted-foreground">{t.description}</p>
                </li>
              ))}
            </ul>
          )}
        </Section>

        <Section title="Daily Summary">
          <p className="text-sm text-muted-foreground">A quick digest of your active tasks for today.</p>
          <Button size="sm" disabled={dailySummary.isPending} onClick={() => dailySummary.mutate()}>
            {dailySummary.isPending ? "Summarizing…" : "Summarize my day"}
          </Button>
          {dailySummary.data && <p className="text-sm">{dailySummary.data.summary}</p>}
        </Section>

        <Section title="Weekly Report Summary">
          <p className="text-sm text-muted-foreground">Narrative summary of the selected project's last 7 days.</p>
          <Button size="sm" disabled={!projectId || weeklySummary.isPending} onClick={() => weeklySummary.mutate()}>
            {weeklySummary.isPending ? "Summarizing…" : "Summarize this week"}
          </Button>
          {weeklySummary.data && <p className="text-sm">{weeklySummary.data.summary}</p>}
        </Section>

        <Section title="Team Productivity Analysis">
          <p className="text-sm text-muted-foreground">Analysis of the selected project's completion trends over the last 30 days.</p>
          <Button size="sm" disabled={!projectId || productivity.isPending} onClick={() => productivity.mutate()}>
            {productivity.isPending ? "Analyzing…" : "Analyze productivity"}
          </Button>
          {productivity.data && <p className="text-sm">{productivity.data.summary}</p>}
        </Section>

        <Section title="Predict Late Tasks">
          <p className="text-sm text-muted-foreground">Flags open tasks in the selected project that look at risk of finishing late.</p>
          <Button size="sm" disabled={!projectId || lateTasks.isPending} onClick={() => lateTasks.mutate()}>
            {lateTasks.isPending ? "Checking…" : "Predict late tasks"}
          </Button>
          {lateTasks.data && (
            <ul className="flex flex-col gap-2">
              {lateTasks.data.at_risk.length === 0 && <li className="text-sm text-muted-foreground">No tasks currently look at risk.</li>}
              {lateTasks.data.at_risk.map((r) => (
                <li key={r.task_id} className="rounded-lg border border-border p-3 text-sm">
                  <span className="font-medium">{r.title}</span>
                  <p className="mt-1 text-muted-foreground">{r.reasoning}</p>
                </li>
              ))}
            </ul>
          )}
        </Section>

        <Section title="Estimate Duration">
          <Input placeholder="Task title" value={estTitle} onChange={(e) => setEstTitle(e.target.value)} />
          <Textarea rows={2} placeholder="Description" value={estDesc} onChange={(e) => setEstDesc(e.target.value)} />
          <Button size="sm" disabled={!estTitle || estimate.isPending} onClick={() => estimate.mutate()}>
            {estimate.isPending ? "Estimating…" : "Estimate hours"}
          </Button>
          {estimate.data && (
            <p className="text-sm">
              <span className="font-medium">{estimate.data.estimate_hours}h</span> — {estimate.data.reasoning}
            </p>
          )}
        </Section>

        <Section title="Suggest Priority">
          <Input placeholder="Task title" value={prTitle} onChange={(e) => setPrTitle(e.target.value)} />
          <Textarea rows={2} placeholder="Description" value={prDesc} onChange={(e) => setPrDesc(e.target.value)} />
          <Button size="sm" disabled={!prTitle || priority.isPending} onClick={() => priority.mutate()}>
            {priority.isPending ? "Thinking…" : "Suggest priority"}
          </Button>
          {priority.data && (
            <p className="text-sm">
              <span className="font-medium capitalize">{priority.data.priority}</span> — {priority.data.reasoning}
            </p>
          )}
        </Section>

        <Section title="Suggest Assignee">
          <Input placeholder="Task title" value={asTitle} onChange={(e) => setAsTitle(e.target.value)} />
          <Textarea rows={2} placeholder="Description" value={asDesc} onChange={(e) => setAsDesc(e.target.value)} />
          <Button size="sm" disabled={!projectId || !asTitle || assignee.isPending} onClick={() => assignee.mutate()}>
            {assignee.isPending ? "Thinking…" : "Suggest assignee"}
          </Button>
          {assignee.data && (
            <p className="text-sm">
              <span className="font-medium">{assignee.data.assignee_name}</span> — {assignee.data.reasoning}
            </p>
          )}
        </Section>

        <Section title="Meeting Summary">
          <Textarea
            rows={5}
            placeholder="Paste raw meeting notes here…"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
          <Button size="sm" disabled={!notes || meeting.isPending} onClick={() => meeting.mutate()}>
            {meeting.isPending ? "Summarizing…" : "Summarize meeting"}
          </Button>
          {meeting.data && (
            <div className="text-sm">
              <p>{meeting.data.summary}</p>
              {meeting.data.action_items.length > 0 && (
                <ul className="mt-2 list-disc pl-5">
                  {meeting.data.action_items.map((item, i) => (
                    <li key={i}>{item}</li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </Section>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit `/ai-assistant`: confirm no "← Dashboard" link remains, all 8 tool sections still submit and render results as before, and triggering a failure (e.g. click "Summarize this week" with no project selected is disabled by the button — instead, stop the backend and click any enabled action) shows the app-wide error toast instead of inline red text.

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/ai/AIAssistantPage.tsx
git commit -m "$(cat <<'EOF'
Migrate AIAssistantPage onto the new shell and design-system primitives

Removes the page-local ErrorText component/call sites — all 8 tool
mutations now surface failures through the app-wide error toast.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Task 18: Migrate NotificationPreferencesPage

**Files:**
- Modify: `src/features/notifications/NotificationPreferencesPage.tsx` (full rewrite, current file is 82 lines)

**Interfaces:**
- Consumes: `Input` (Task 2). Drops the "← Dashboard" back-link but keeps its own `<h1>Notification Settings</h1>`. Native checkboxes are left as plain `<input type="checkbox">` — no design-system Checkbox primitive was in scope (Task 2's component list) since this is the only place checkboxes are used in the app; adding one now would be an unrequested abstraction for a single call site.

- [ ] **Step 1: Replace the full contents of `src/features/notifications/NotificationPreferencesPage.tsx`**

```tsx
import { useEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { getPreference, updatePreference } from "@/features/notifications/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export default function NotificationPreferencesPage() {
  const { data } = useQuery({ queryKey: ["notification-preference"], queryFn: getPreference });

  const [emailEnabled, setEmailEnabled] = useState(true);
  const [lineEnabled, setLineEnabled] = useState(false);
  const [lineToken, setLineToken] = useState("");

  useEffect(() => {
    if (!data) return;
    setEmailEnabled(data.email_enabled);
    setLineEnabled(data.line_enabled);
    setLineToken(data.line_notify_token ?? "");
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: () => updatePreference({ email_enabled: emailEnabled, line_enabled: lineEnabled, line_notify_token: lineToken }),
  });

  return (
    <div className="p-6">
      <h1 className="mb-6 text-2xl font-semibold">Notification Settings</h1>

      <Card className="max-w-md">
        <CardHeader>
          <CardTitle>Delivery channels</CardTitle>
        </CardHeader>
        <CardContent>
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              saveMutation.mutate();
            }}
          >
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={emailEnabled} onChange={(e) => setEmailEnabled(e.target.checked)} />
              Email notifications
            </label>

            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={lineEnabled} onChange={(e) => setLineEnabled(e.target.checked)} />
              LINE Notify
            </label>

            {lineEnabled && (
              <div className="flex flex-col gap-1">
                <Input placeholder="LINE Notify token" value={lineToken} onChange={(e) => setLineToken(e.target.value)} />
                <p className="text-xs text-muted-foreground">
                  Get a personal token from{" "}
                  <a href="https://notify-bot.line.me/my/" target="_blank" rel="noreferrer" className="underline">
                    notify-bot.line.me
                  </a>
                </p>
              </div>
            )}

            <Button type="submit" disabled={saveMutation.isPending} className="self-start">
              {saveMutation.isPending ? "Saving…" : "Save"}
            </Button>
            {saveMutation.isSuccess && <p className="text-sm text-primary">Saved.</p>}
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
```

- [ ] **Step 2: Verify**

Run: `npm run build && npm run lint`
Expected: both exit 0.

`npm run dev`, visit `/settings/notifications` (via the top-bar avatar menu's "Notification settings" link, Task 4): confirm no "← Dashboard" link remains, toggling LINE Notify still reveals the token field, and saving still shows "Saved."

- [ ] **Step 3: Commit**

```bash
cd /Users/cms-rd-1/Documents/GitHub/Taskworked
git add frontend/src/features/notifications/NotificationPreferencesPage.tsx
git commit -m "$(cat <<'EOF'
Migrate NotificationPreferencesPage onto the new shell and Input primitive

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

## Final verification (after all 18 tasks)

- [ ] Run `npm run build && npm run lint` one more time from a clean state — both exit 0.
- [ ] Full manual walkthrough with the backend + infra running (`make infra`, `make dev`): log in → dashboard → toggle dark mode (persists on reload) → collapse/expand sidebar (persists on reload) → open a project → click through all 6 tabs → open a task, add a comment/checklist item/attachment/tag, watch/unwatch it → drag a card on the Board → drag an event on the Calendar → check Gantt renders and a dependency can be added/removed → Reports Weekly/Monthly toggle + Export CSV + Save as PDF (print preview hides nav) → Team directory role/department edits (as admin) → Action Plan goal/milestone/task-assignment flow → Gamification leaderboard toggle → AI Assistant: run at least 2 of the 8 tools → Notification Settings save → trigger one deliberate failure (e.g. stop the backend, submit any form) and confirm the error toast appears → sign out via the top-bar user menu.
- [ ] Confirm no page contains `min-h-screen` or a `← <Something>` back-link (`grep -rn "min-h-screen\|← " frontend/src/features` should return nothing).
