import { api } from "@/lib/api";
import type { Leaderboard, Profile } from "@/features/gamification/types";

export function getProfile() {
  return api.get<{ data: Profile }>("/gamification/profile").then((r) => r.data.data);
}

export function getLeaderboard() {
  return api.get<{ data: Leaderboard }>("/gamification/leaderboard").then((r) => r.data.data);
}
