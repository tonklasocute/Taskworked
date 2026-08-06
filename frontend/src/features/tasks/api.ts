import { api } from "@/lib/api";
import type { Task, TaskListResponse, TaskPriority, TaskStatus } from "@/features/tasks/types";

export function listTasks(projectId: string) {
  return api
    .get<{ data: TaskListResponse }>("/tasks", { params: { project_id: projectId } })
    .then((r) => r.data.data);
}

export function createTask(input: { project_id: string; title: string; description?: string; priority?: TaskPriority }) {
  return api.post<{ data: Task }>("/tasks", input).then((r) => r.data.data);
}

export function updateTaskStatus(id: string, status: TaskStatus) {
  return api.patch<{ data: Task }>(`/tasks/${id}`, { status }).then((r) => r.data.data);
}
