import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { DndContext, useDraggable, useDroppable, type DragEndEvent } from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";
import { getProject } from "@/features/projects/api";
import { listTasks, updateTaskStatus } from "@/features/tasks/api";
import { useTaskSocket } from "@/features/tasks/useTaskSocket";
import type { Task, TaskPriority, TaskStatus } from "@/features/tasks/types";
import { Card, CardContent } from "@/components/ui/card";

const COLUMNS: { status: TaskStatus; label: string }[] = [
  { status: "backlog", label: "Backlog" },
  { status: "todo", label: "Todo" },
  { status: "doing", label: "Doing" },
  { status: "review", label: "Review" },
  { status: "testing", label: "Testing" },
  { status: "done", label: "Done" },
  { status: "blocked", label: "Blocked" },
];

const PRIORITY_COLOR: Record<TaskPriority, string> = {
  critical: "bg-destructive text-destructive-foreground",
  high: "bg-primary text-primary-foreground",
  medium: "bg-muted text-muted-foreground",
  low: "bg-muted text-muted-foreground",
};

export default function KanbanBoard() {
  const { projectId } = useParams<{ projectId: string }>();
  const queryClient = useQueryClient();

  const { data: project } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => getProject(projectId!),
    enabled: !!projectId,
  });

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
    <div className="min-h-screen p-6">
      <div className="mb-6">
        <Link to={`/projects/${projectId}`} className="text-sm text-muted-foreground">← List view</Link>
        <h1 className="text-2xl font-semibold">{project?.name ?? "Board"}</h1>
      </div>

      <DndContext onDragEnd={handleDragEnd}>
        <div className="flex gap-4 overflow-x-auto pb-4">
          {COLUMNS.map((col) => (
            <BoardColumn
              key={col.status}
              status={col.status}
              label={col.label}
              tasks={tasks?.items.filter((t) => t.status === col.status) ?? []}
            />
          ))}
        </div>
      </DndContext>
    </div>
  );
}

function BoardColumn({ status, label, tasks }: { status: TaskStatus; label: string; tasks: Task[] }) {
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
        <TaskCard key={task.id} task={task} />
      ))}
    </div>
  );
}

function TaskCard({ task }: { task: Task }) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: task.id });

  const style = transform
    ? { transform: CSS.Translate.toString(transform), zIndex: 10 }
    : undefined;

  return (
    <Card
      ref={setNodeRef}
      style={style}
      {...listeners}
      {...attributes}
      className={`cursor-grab touch-none active:cursor-grabbing ${isDragging ? "opacity-50" : ""}`}
    >
      <CardContent className="p-3">
        <p className="text-sm font-medium">{task.title}</p>
        <div className="mt-2 flex flex-wrap items-center gap-1">
          <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${PRIORITY_COLOR[task.priority]}`}>
            {task.priority}
          </span>
          {task.tags.map((tag) => (
            <span key={tag} className="inline-flex rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
              {tag}
            </span>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
