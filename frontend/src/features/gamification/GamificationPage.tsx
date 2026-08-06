import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { getLeaderboard, getProfile } from "@/features/gamification/api";
import type { BadgeCode, MissionType } from "@/features/gamification/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

const BADGE_INFO: Record<BadgeCode, { label: string; emoji: string }> = {
  early_bird: { label: "Early Bird", emoji: "🐦" },
  fast_worker: { label: "Fast Worker", emoji: "⚡" },
  hundred_tasks: { label: "100 Tasks", emoji: "💯" },
  perfect_week: { label: "Perfect Week", emoji: "✨" },
  seven_day_streak: { label: "7 Day Streak", emoji: "🔥" },
  legend: { label: "Legend", emoji: "👑" },
};

const LEVEL_EMOJI: Record<string, string> = {
  Novice: "🌱",
  Adept: "🌿",
  Master: "🌳",
  Legend: "🏆",
};

const MISSION_LABEL: Record<MissionType, string> = {
  daily: "Daily Mission",
  weekly: "Weekly Mission",
  monthly: "Monthly Mission",
};

export default function GamificationPage() {
  const [leaderboardTab, setLeaderboardTab] = useState<"individual" | "department">("individual");

  const { data: profile, isLoading } = useQuery({ queryKey: ["gamification-profile"], queryFn: getProfile });
  const { data: leaderboard } = useQuery({ queryKey: ["gamification-leaderboard"], queryFn: getLeaderboard });

  return (
    <div className="min-h-screen p-6">
      <div className="mb-6">
        <Link to="/dashboard" className="text-sm text-muted-foreground">← Dashboard</Link>
        <h1 className="text-2xl font-semibold">Gamification</h1>
      </div>

      {isLoading && <p className="text-muted-foreground">Loading profile…</p>}

      {profile && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="flex flex-col gap-6 lg:col-span-2">
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center gap-4">
                  <div className="flex h-16 w-16 items-center justify-center rounded-full bg-primary/10 text-3xl">
                    {LEVEL_EMOJI[profile.character.level_title] ?? "🌱"}
                  </div>
                  <div className="flex-1">
                    <p className="text-lg font-semibold">
                      Level {profile.character.level} — {profile.character.level_title}
                    </p>
                    <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full bg-primary transition-all"
                        style={{ width: `${(profile.character.exp_into_level / profile.character.exp_per_level) * 100}%` }}
                      />
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {profile.character.exp_into_level} / {profile.character.exp_per_level} EXP to next level · {profile.character.exp} total EXP
                    </p>
                  </div>
                </div>
                <div className="mt-4 grid grid-cols-3 gap-3 text-center">
                  <div>
                    <p className="text-xl font-semibold">{profile.character.total_completed}</p>
                    <p className="text-xs text-muted-foreground">Tasks completed</p>
                  </div>
                  <div>
                    <p className="text-xl font-semibold">🔥 {profile.character.current_streak}</p>
                    <p className="text-xs text-muted-foreground">Current streak</p>
                  </div>
                  <div>
                    <p className="text-xl font-semibold">{profile.character.longest_streak}</p>
                    <p className="text-xs text-muted-foreground">Longest streak</p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Badges</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-3 gap-3 sm:grid-cols-6">
                  {(Object.keys(BADGE_INFO) as BadgeCode[]).map((code) => {
                    const earned = profile.badges.some((b) => b.code === code);
                    const info = BADGE_INFO[code];
                    return (
                      <div
                        key={code}
                        title={info.label}
                        className={`flex flex-col items-center gap-1 rounded-lg border border-border p-3 text-center ${earned ? "" : "opacity-30 grayscale"}`}
                      >
                        <span className="text-2xl">{info.emoji}</span>
                        <span className="text-[10px] leading-tight">{info.label}</span>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Missions</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-4">
                {profile.missions.map((m) => (
                  <div key={m.type}>
                    <div className="mb-1 flex items-center justify-between text-sm">
                      <span>{MISSION_LABEL[m.type]}</span>
                      <span className="text-muted-foreground">
                        {m.count}/{m.threshold} {m.completed && "✓"} · +{m.reward} EXP
                      </span>
                    </div>
                    <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                      <div
                        className={`h-full transition-all ${m.completed ? "bg-primary" : "bg-primary/60"}`}
                        style={{ width: `${Math.min(100, (m.count / m.threshold) * 100)}%` }}
                      />
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between">
              <CardTitle>Leaderboard</CardTitle>
              <div className="flex gap-1 rounded-lg border border-border p-1">
                <Button size="sm" variant={leaderboardTab === "individual" ? "default" : "ghost"} onClick={() => setLeaderboardTab("individual")}>
                  People
                </Button>
                <Button size="sm" variant={leaderboardTab === "department" ? "default" : "ghost"} onClick={() => setLeaderboardTab("department")}>
                  Departments
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {leaderboardTab === "individual" ? (
                <ol className="flex flex-col gap-2">
                  {leaderboard?.individuals.map((entry, i) => (
                    <li key={entry.user_id} className="flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2">
                        <span className="w-5 text-muted-foreground">{i + 1}.</span>
                        {entry.name}
                      </span>
                      <span className="text-muted-foreground">Lv.{entry.level} · {entry.exp} EXP</span>
                    </li>
                  ))}
                  {leaderboard?.individuals.length === 0 && <p className="text-sm text-muted-foreground">No activity yet.</p>}
                </ol>
              ) : (
                <ol className="flex flex-col gap-2">
                  {leaderboard?.departments.map((entry, i) => (
                    <li key={entry.department_id} className="flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2">
                        <span className="w-5 text-muted-foreground">{i + 1}.</span>
                        {entry.department_name}
                      </span>
                      <span className="text-muted-foreground">{entry.total_exp} EXP · {entry.member_count} members</span>
                    </li>
                  ))}
                  {leaderboard?.departments.length === 0 && <p className="text-sm text-muted-foreground">No departments with activity yet.</p>}
                </ol>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
