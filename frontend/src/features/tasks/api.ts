import { api } from "@/lib/api";
import type { Task, TaskListResponse, TaskPriority, TaskStatus } from "@/features/tasks/types";

export function listTasks(projectId: string, filters?: { status?: TaskStatus; assignee_id?: string }) {
  return api
    .get<{ data: TaskListResponse }>("/tasks", { params: { project_id: projectId, ...filters } })
    .then((r) => r.data.data);
}

export function createTask(input: { project_id: string; title: string; description?: string; priority?: TaskPriority }) {
  return api.post<{ data: Task }>("/tasks", input).then((r) => r.data.data);
}

export function updateTaskStatus(id: string, status: TaskStatus) {
  return api.patch<{ data: Task }>(`/tasks/${id}`, { status }).then((r) => r.data.data);
}

export function updateTaskDates(id: string, dates: { start_date?: string; due_date?: string }) {
  return api.patch<{ data: Task }>(`/tasks/${id}`, dates).then((r) => r.data.data);
}
