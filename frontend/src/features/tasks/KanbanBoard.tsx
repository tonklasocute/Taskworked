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
