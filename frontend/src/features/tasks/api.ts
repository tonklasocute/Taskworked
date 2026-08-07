import { api } from "@/lib/api";
import type { GanttView, MySummary, Task, TaskListResponse, TaskPriority, TaskStatus } from "@/features/tasks/types";

export function listTasks(projectId: string, filters?: { status?: TaskStatus; assignee_id?: string }) {
  return api
    .get<{ data: TaskListResponse }>("/tasks", { params: { project_id: projectId, ...filters } })
    .then((r) => r.data.data);
}

export function getMySummary() {
  return api.get<{ data: MySummary }>("/tasks/my-summary").then((r) => r.data.data);
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

export function getGanttView(projectId: string) {
  return api.get<{ data: GanttView }>(`/projects/${projectId}/gantt`).then((r) => r.data.data);
}

export function addDependency(taskId: string, dependsOnTaskId: string) {
  return api.post(`/tasks/${taskId}/dependencies`, { depends_on_task_id: dependsOnTaskId });
}

export function removeDependency(taskId: string, dependsOnTaskId: string) {
  return api.delete(`/tasks/${taskId}/dependencies/${dependsOnTaskId}`);
}
