import type { Task } from "@/features/tasks/types";

export type GoalStatus = "not_started" | "in_progress" | "completed";

export interface Goal {
  id: string;
  project_id: string;
  name: string;
  description: string;
  status: GoalStatus;
  due_date?: string;
  created_at: string;
}

export interface Milestone {
  id: string;
  goal_id: string;
  name: string;
  description: string;
  status: GoalStatus;
  due_date?: string;
  created_at: string;
}

// The backend embeds task.Response / MilestoneResponse / GoalResponse
// directly (Go struct embedding flattens into the parent JSON object), so
// these extend the flat shape rather than nesting under a sub-key.
export interface TaskNode extends Task {
  subtasks: Task[];
}

export interface MilestoneNode extends Milestone {
  tasks: TaskNode[];
}

export interface GoalNode extends Goal {
  milestones: MilestoneNode[];
}

export interface ActionPlanView {
  goals: GoalNode[];
}
