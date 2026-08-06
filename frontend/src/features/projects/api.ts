import { api } from "@/lib/api";
import type { Project, ProjectListResponse, ProjectMember } from "@/features/projects/types";

export function listProjects() {
  return api.get<{ data: ProjectListResponse }>("/projects").then((r) => r.data.data);
}

export function createProject(input: { name: string; description?: string; color?: string; icon?: string }) {
  return api.post<{ data: Project }>("/projects", input).then((r) => r.data.data);
}

export function getProject(id: string) {
  return api.get<{ data: Project }>(`/projects/${id}`).then((r) => r.data.data);
}

export function getProjectMembers(id: string) {
  return api.get<{ data: ProjectMember[] }>(`/projects/${id}/members`).then((r) => r.data.data);
}
