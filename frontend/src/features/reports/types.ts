import type { Task } from "@/features/tasks/types";

export interface ReportSummary {
  total_tasks: number;
  created_in_period: number;
  completed_in_period: number;
  completion_rate: number;
  overdue_count: number;
  near_due_count: number;
}

export interface AssigneePerformance {
  assignee_id: string;
  completed_count: number;
  on_time_count: number;
  late_count: number;
  on_time_rate: number;
}

export interface ProductivityPoint {
  date: string;
  completed_count: number;
}

export interface PeriodReport {
  from: string;
  to: string;
  summary: ReportSummary;
  late_tasks: Task[];
  performance: AssigneePerformance[];
  productivity: ProductivityPoint[];
}
