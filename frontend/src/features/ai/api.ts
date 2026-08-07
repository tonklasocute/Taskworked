import { api } from "@/lib/api";
import type {
  AssigneeSuggestion,
  EstimateResult,
  LateTaskRisk,
  MeetingSummaryResult,
  PriorityResult,
  SummaryResult,
  TaskSuggestion,
} from "@/features/ai/types";

export function generateTasks(input: { project_id: string; prompt: string }) {
  return api.post<{ data: { tasks: TaskSuggestion[] } }>("/ai/generate-tasks", input).then((r) => r.data.data);
}

export function getDailySummary() {
  return api.get<{ data: SummaryResult }>("/ai/daily-summary").then((r) => r.data.data);
}

export function getWeeklySummary(projectId: string) {
  return api.get<{ data: SummaryResult }>(`/ai/projects/${projectId}/weekly-summary`).then((r) => r.data.data);
}

export function estimateDuration(input: { title: string; description: string }) {
  return api.post<{ data: EstimateResult }>("/ai/estimate-duration", input).then((r) => r.data.data);
}

export function suggestPriority(input: { title: string; description: string }) {
  return api.post<{ data: PriorityResult }>("/ai/suggest-priority", input).then((r) => r.data.data);
}

export function suggestAssignee(input: { project_id: string; title: string; description: string }) {
  return api.post<{ data: AssigneeSuggestion }>("/ai/suggest-assignee", input).then((r) => r.data.data);
}

export function predictLateTasks(projectId: string) {
  return api.get<{ data: { at_risk: LateTaskRisk[] } }>(`/ai/projects/${projectId}/predict-late`).then((r) => r.data.data);
}

export function getProductivityAnalysis(projectId: string) {
  return api.get<{ data: SummaryResult }>(`/ai/projects/${projectId}/productivity`).then((r) => r.data.data);
}

export function generateMeetingSummary(input: { notes: string }) {
  return api.post<{ data: MeetingSummaryResult }>("/ai/meeting-summary", input).then((r) => r.data.data);
}
