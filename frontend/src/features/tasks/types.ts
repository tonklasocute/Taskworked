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
  milestone_id?: string;
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

export interface MySummary {
  active_count: number;
  overdue_count: number;
  due_soon_count: number;
  tasks: Task[];
}

export interface TaskComment {
  id: string;
  task_id: string;
  author_id: string;
  author_name: string;
  body: string;
  created_at: string;
}

export interface TaskAttachment {
  id: string;
  task_id: string;
  uploader_id: string;
  uploader_name: string;
  file_name: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
}

export interface TaskWatcher {
  user_id: string;
  name: string;
  email: string;
}

export interface WatchStatus {
  watching: boolean;
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
