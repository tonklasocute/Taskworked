export type BadgeCode = "early_bird" | "fast_worker" | "hundred_tasks" | "legend" | "perfect_week" | "seven_day_streak";
export type MissionType = "daily" | "weekly" | "monthly";

export interface Character {
  level: number;
  level_title: string;
  exp: number;
  exp_into_level: number;
  exp_per_level: number;
  total_completed: number;
  current_streak: number;
  longest_streak: number;
}

export interface Badge {
  code: BadgeCode;
  awarded_at: string;
}

export interface Mission {
  type: MissionType;
  count: number;
  threshold: number;
  reward: number;
  completed: boolean;
}

export interface Profile {
  character: Character;
  badges: Badge[];
  missions: Mission[];
}

export interface LeaderboardEntry {
  user_id: string;
  name: string;
  level: number;
  exp: number;
}

export interface DepartmentLeaderboardEntry {
  department_id: string;
  department_name: string;
  total_exp: number;
  member_count: number;
}

export interface Leaderboard {
  individuals: LeaderboardEntry[];
  departments: DepartmentLeaderboardEntry[];
}
