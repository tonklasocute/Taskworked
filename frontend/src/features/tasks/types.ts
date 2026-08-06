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
