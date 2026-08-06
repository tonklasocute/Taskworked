export type ProjectStatus = "planning" | "active" | "on_hold" | "completed" | "archived";

export interface Project {
  id: string;
  name: string;
  description: string;
  owner_id: string;
  status: ProjectStatus;
  due_date?: string;
  color: string;
  icon: string;
  created_at: string;
}

export interface ProjectListResponse {
  items: Project[];
  total: number;
  page: number;
  page_size: number;
}
