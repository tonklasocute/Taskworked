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
