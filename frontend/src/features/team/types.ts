export type OrgRole = "super_admin" | "admin" | "manager" | "leader" | "employee";

export interface Department {
  id: string;
  name: string;
  created_at: string;
}

export interface DirectoryEntry {
  user_id: string;
  name: string;
  email: string;
  role: OrgRole;
  avatar_url: string;
  department_id?: string;
  department_name?: string;
  online: boolean;
  workload: number;
}

export interface Directory {
  members: DirectoryEntry[];
  departments: Department[];
}
