import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import FullCalendar from "@fullcalendar/react";
import dayGridPlugin from "@fullcalendar/daygrid";
import timeGridPlugin from "@fullcalendar/timegrid";
import interactionPlugin from "@fullcalendar/interaction";
import type { EventDropArg } from "@fullcalendar/core";
import type { EventResizeDoneArg } from "@fullcalendar/interaction";
import { getProject } from "@/features/projects/api";
import { listTasks, updateTaskDates } from "@/features/tasks/api";
import type { Task, TaskPriority, TaskStatus } from "@/features/tasks/types";

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

// FullCalendar's all-day `end` is exclusive, but our due_date is inclusive
// (the day the task is due) — so events get +1 day on the way in and -1 on
// the way back out.
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

  const { data: project } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => getProject(projectId!),
    enabled: !!projectId,
  });

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
    <div className="min-h-screen p-6">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <Link to={`/projects/${projectId}`} className="text-sm text-muted-foreground">← List view</Link>
          <h1 className="text-2xl font-semibold">{project?.name ?? "Calendar"}</h1>
        </div>
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value as TaskStatus | "")}
          className="h-10 rounded-lg border border-border bg-transparent px-3 text-sm"
        >
          <option value="">All statuses</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
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
