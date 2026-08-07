export interface TaskSuggestion {
  title: string;
  description: string;
  priority: "critical" | "high" | "medium" | "low";
  estimate_hours: number;
}

export interface SummaryResult {
  summary: string;
}

export interface EstimateResult {
  estimate_hours: number;
  reasoning: string;
}

export interface PriorityResult {
  priority: "critical" | "high" | "medium" | "low";
  reasoning: string;
}

export interface AssigneeSuggestion {
  assignee_id: string;
  assignee_name: string;
  reasoning: string;
}

export interface LateTaskRisk {
  task_id: string;
  title: string;
  reasoning: string;
}

export interface MeetingSummaryResult {
  summary: string;
  action_items: string[];
}
