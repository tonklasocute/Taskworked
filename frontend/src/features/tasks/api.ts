import { api } from "@/lib/api";
import type {
  GanttView,
  MySummary,
  Task,
  TaskAttachment,
  TaskComment,
  TaskListResponse,
  TaskPriority,
  TaskStatus,
  TaskWatcher,
  WatchStatus,
} from "@/features/tasks/types";

export function listTasks(projectId: string, filters?: { status?: TaskStatus; assignee_id?: string }) {
  return api
    .get<{ data: TaskListResponse }>("/tasks", { params: { project_id: projectId, ...filters } })
    .then((r) => r.data.data);
}

export function getTask(id: string) {
  return api.get<{ data: Task }>(`/tasks/${id}`).then((r) => r.data.data);
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

export function listComments(taskId: string) {
  return api.get<{ data: TaskComment[] }>(`/tasks/${taskId}/comments`).then((r) => r.data.data ?? []);
}

export function addComment(taskId: string, body: string) {
  return api.post<{ data: TaskComment }>(`/tasks/${taskId}/comments`, { body }).then((r) => r.data.data);
}

export function deleteComment(taskId: string, commentId: string) {
  return api.delete(`/tasks/${taskId}/comments/${commentId}`);
}

export function listAttachments(taskId: string) {
  return api.get<{ data: TaskAttachment[] }>(`/tasks/${taskId}/attachments`).then((r) => r.data.data ?? []);
}

export function uploadAttachment(taskId: string, file: File) {
  const form = new FormData();
  form.append("file", file);
  return api.post<{ data: TaskAttachment }>(`/tasks/${taskId}/attachments`, form).then((r) => r.data.data);
}

export function getAttachmentDownloadURL(taskId: string, attachmentId: string) {
  return api.get<{ data: { url: string } }>(`/tasks/${taskId}/attachments/${attachmentId}/download`).then((r) => r.data.data.url);
}

export function deleteAttachment(taskId: string, attachmentId: string) {
  return api.delete(`/tasks/${taskId}/attachments/${attachmentId}`);
}

export function getWatchStatus(taskId: string) {
  return api.get<{ data: WatchStatus }>(`/tasks/${taskId}/watch`).then((r) => r.data.data);
}

export function watchTask(taskId: string) {
  return api.post(`/tasks/${taskId}/watch`);
}

export function unwatchTask(taskId: string) {
  return api.delete(`/tasks/${taskId}/watch`);
}

export function listWatchers(taskId: string) {
  return api.get<{ data: TaskWatcher[] }>(`/tasks/${taskId}/watchers`).then((r) => r.data.data ?? []);
}
