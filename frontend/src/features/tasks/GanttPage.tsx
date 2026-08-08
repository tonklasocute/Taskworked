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
