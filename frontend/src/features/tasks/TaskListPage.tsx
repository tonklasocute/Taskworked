import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { createTask, listTasks, updateTaskStatus } from "@/features/tasks/api";
import { getProject } from "@/features/projects/api";
import type { TaskPriority, TaskStatus } from "@/features/tasks/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

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
const PRIORITY_COLOR: Record<TaskPriority, string> = {
  critical: "bg-destructive text-destructive-foreground",
  high: "bg-primary text-primary-foreground",
  medium: "bg-muted text-muted-foreground",
  low: "bg-muted text-muted-foreground",
};

export default function TaskListPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [title, setTitle] = useState("");
  const [priority, setPriority] = useState<TaskPriority>("medium");

  const { data: project } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => getProject(projectId!),
    enabled: !!projectId,
  });

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
    <div className="min-h-screen p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <Link to="/projects" className="text-sm text-muted-foreground">← Projects</Link>
          <h1 className="text-2xl font-semibold">{project?.name ?? "Tasks"}</h1>
        </div>
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
              <input
                required
                placeholder="Task title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="h-10 flex-1 rounded-lg border border-border bg-transparent px-3 text-sm"
              />
              <select
                value={priority}
                onChange={(e) => setPriority(e.target.value as TaskPriority)}
                className="h-10 rounded-lg border border-border bg-transparent px-3 text-sm"
              >
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
              </select>
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? "Creating…" : "Create task"}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {isLoading && <p className="text-muted-foreground">Loading tasks…</p>}

      {!isLoading && tasks?.items.length === 0 && (
        <p className="text-muted-foreground">No tasks yet. Create your first one above.</p>
      )}

      <div className="flex flex-col gap-3">
        {tasks?.items.map((t) => (
          <Card key={t.id}>
            <CardContent className="flex flex-wrap items-center justify-between gap-3 py-4">
              <div>
                <p className="font-medium">{t.title}</p>
                <span className={`mt-1 inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${PRIORITY_COLOR[t.priority]}`}>
                  {t.priority}
                </span>
              </div>
              <select
                value={t.status}
                onChange={(e) => statusMutation.mutate({ id: t.id, status: e.target.value as TaskStatus })}
                className="h-9 rounded-lg border border-border bg-transparent px-2 text-sm"
              >
                {STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {STATUS_LABEL[s]}
                  </option>
                ))}
              </select>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
