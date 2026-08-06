import { api } from "@/lib/api";
import type { Project, ProjectListResponse } from "@/features/projects/types";

export function listProjects() {
  return api.get<{ data: ProjectListResponse }>("/projects").then((r) => r.data.data);
}

export function createProject(input: { name: string; description?: string; color?: string; icon?: string }) {
  return api.post<{ data: Project }>("/projects", input).then((r) => r.data.data);
}
