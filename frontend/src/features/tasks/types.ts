export type TaskPriority = "critical" | "high" | "medium" | "low";
export type TaskStatus = "backlog" | "todo" | "doing" | "review" | "testing" | "done" | "blocked";

export interface Task {
  id: string;
  project_id: string;
  parent_task_id?: string;
  title: string;
  description: string;
  priority: TaskPriority;
  status: TaskStatus;
  start_date?: string;
  due_date?: string;
  estimate_hours?: number;
  assignee_id?: string;
  reporter_id: string;
  tags: string[];
  created_at: string;
}

export interface TaskListResponse {
  items: Task[];
  total: number;
  page: number;
  page_size: number;
}

export interface GanttDependency {
  task_id: string;
  depends_on_task_id: string;
}

export interface GanttView {
  tasks: Task[];
  dependencies: GanttDependency[];
  critical_path: string[];
  project_days: number;
}
