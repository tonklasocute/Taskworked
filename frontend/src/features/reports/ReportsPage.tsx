import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import { getProject, getProjectMembers } from "@/features/projects/api";
import { downloadCsv, getPeriodReport } from "@/features/reports/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

function StatTile({ label, value }: { label: string; value: string | number }) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="mt-1 text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}

export default function ReportsPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const [period, setPeriod] = useState<"week" | "month">("week");

  const { data: project } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => getProject(projectId!),
    enabled: !!projectId,
  });

  const { data: report, isLoading } = useQuery({
    queryKey: ["report", projectId, period],
    queryFn: () => getPeriodReport(projectId!, period),
    enabled: !!projectId,
  });

  const { data: members } = useQuery({
    queryKey: ["project-members", projectId],
    queryFn: () => getProjectMembers(projectId!),
    enabled: !!projectId,
  });
  const nameByUserId = new Map((members ?? []).map((m) => [m.user_id, m.name]));

  return (
    <div className="min-h-screen p-6 print:p-0">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3 print:hidden">
        <div>
          <Link to={`/projects/${projectId}`} className="text-sm text-muted-foreground">← List view</Link>
          <h1 className="text-2xl font-semibold">{project?.name ?? "Reports"}</h1>
        </div>
        <div className="flex gap-2">
          <div className="flex gap-1 rounded-lg border border-border p-1">
            <Button size="sm" variant={period === "week" ? "default" : "ghost"} onClick={() => setPeriod("week")}>
              Weekly
            </Button>
            <Button size="sm" variant={period === "month" ? "default" : "ghost"} onClick={() => setPeriod("month")}>
              Monthly
            </Button>
          </div>
          <Button variant="outline" onClick={() => downloadCsv(projectId!)}>Export CSV</Button>
          <Button variant="outline" onClick={() => window.print()}>Save as PDF</Button>
        </div>
      </div>

      {isLoading && <p className="text-muted-foreground">Loading report…</p>}

      {report && (
        <div className="flex flex-col gap-6">
          <h2 className="hidden text-xl font-semibold print:block">
            {project?.name} — {period === "week" ? "Weekly" : "Monthly"} Report ({report.from} to {report.to})
          </h2>

          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
            <StatTile label="Total Tasks" value={report.summary.total_tasks} />
            <StatTile label="Created" value={report.summary.created_in_period} />
            <StatTile label="Completed" value={report.summary.completed_in_period} />
            <StatTile label="Completion Rate" value={`${Math.round(report.summary.completion_rate * 100)}%`} />
            <StatTile label="Overdue" value={report.summary.overdue_count} />
            <StatTile label="Near Due" value={report.summary.near_due_count} />
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Productivity</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={report.productivity}>
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                    <XAxis dataKey="date" tick={{ fontSize: 12 }} />
                    <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                    <Tooltip />
                    <Bar dataKey="completed_count" name="Completed" fill="hsl(var(--primary))" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Performance by Assignee</CardTitle>
            </CardHeader>
            <CardContent>
              {report.performance.length === 0 ? (
                <p className="text-sm text-muted-foreground">No tasks completed in this period yet.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border text-left text-xs text-muted-foreground">
                        <th className="py-2 pr-4">Assignee</th>
                        <th className="py-2 pr-4">Completed</th>
                        <th className="py-2 pr-4">On Time</th>
                        <th className="py-2 pr-4">Late</th>
                        <th className="py-2 pr-4">On-Time Rate</th>
                      </tr>
                    </thead>
                    <tbody>
                      {report.performance.map((p) => (
                        <tr key={p.assignee_id} className="border-b border-border last:border-0">
                          <td className="py-2 pr-4">{nameByUserId.get(p.assignee_id) ?? p.assignee_id}</td>
                          <td className="py-2 pr-4">{p.completed_count}</td>
                          <td className="py-2 pr-4">{p.on_time_count}</td>
                          <td className="py-2 pr-4">{p.late_count}</td>
                          <td className="py-2 pr-4">{Math.round(p.on_time_rate * 100)}%</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Late Tasks ({report.late_tasks.length})</CardTitle>
            </CardHeader>
            <CardContent>
              {report.late_tasks.length === 0 ? (
                <p className="text-sm text-muted-foreground">Nothing overdue. 🎉</p>
              ) : (
                <ul className="flex flex-col gap-1">
                  {report.late_tasks.map((t) => (
                    <li key={t.id} className="flex items-center justify-between text-sm">
                      <span>{t.title}</span>
                      <span className="text-destructive">Due {t.due_date}</span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
