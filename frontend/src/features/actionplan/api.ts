import { api } from "@/lib/api";
import type { ActionPlanView, Goal, Milestone } from "@/features/actionplan/types";

export function getActionPlan(projectId: string) {
  return api.get<{ data: ActionPlanView }>(`/projects/${projectId}/action-plan`).then((r) => r.data.data);
}

export function createGoal(input: { project_id: string; name: string; description?: string; due_date?: string }) {
  return api.post<{ data: Goal }>("/goals", input).then((r) => r.data.data);
}

export function createMilestone(input: { goal_id: string; name: string; description?: string; due_date?: string }) {
  return api.post<{ data: Milestone }>("/milestones", input).then((r) => r.data.data);
}

export function assignTaskToMilestone(taskId: string, milestoneId: string) {
  return api.patch(`/tasks/${taskId}`, { milestone_id: milestoneId });
}
