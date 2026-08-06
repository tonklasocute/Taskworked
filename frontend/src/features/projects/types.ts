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

export interface ProjectMember {
  user_id: string;
  role: "owner" | "member";
  name: string;
  email: string;
  avatar_url: string;
}
