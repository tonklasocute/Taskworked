import { api } from "@/lib/api";
import type { Department, Directory, OrgRole } from "@/features/team/types";

export function getDirectory() {
  return api.get<{ data: Directory }>("/team").then((r) => r.data.data);
}

export function createDepartment(name: string) {
  return api.post<{ data: Department }>("/departments", { name }).then((r) => r.data.data);
}

export function deleteDepartment(id: string) {
  return api.delete(`/departments/${id}`);
}

export function updateUserRole(userId: string, role: OrgRole) {
  return api.patch(`/users/${userId}/role`, { role });
}

export function updateUserDepartment(userId: string, departmentId: string | null) {
  return api.patch(`/users/${userId}/department`, { department_id: departmentId });
}
