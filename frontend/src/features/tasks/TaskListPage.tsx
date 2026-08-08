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
