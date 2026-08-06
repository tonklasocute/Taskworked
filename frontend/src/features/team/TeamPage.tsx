import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { useAuthStore } from "@/stores/auth-store";
import { createDepartment, getDirectory, updateUserDepartment, updateUserRole } from "@/features/team/api";
import type { DirectoryEntry, OrgRole } from "@/features/team/types";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

const ROLES: OrgRole[] = ["super_admin", "admin", "manager", "leader", "employee"];

export default function TeamPage() {
  const currentUser = useAuthStore((s) => s.user);
  const isAdmin = currentUser?.role === "super_admin" || currentUser?.role === "admin";
  const queryClient = useQueryClient();
  const [showDeptForm, setShowDeptForm] = useState(false);
  const [deptName, setDeptName] = useState("");

  const { data: directory, isLoading } = useQuery({ queryKey: ["team"], queryFn: getDirectory });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["team"] });

  const deptMutation = useMutation({
    mutationFn: () => createDepartment(deptName),
    onSuccess: () => {
      invalidate();
      setDeptName("");
      setShowDeptForm(false);
    },
  });

  const roleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: OrgRole }) => updateUserRole(userId, role),
    onSuccess: invalidate,
  });

  const deptAssignMutation = useMutation({
    mutationFn: ({ userId, departmentId }: { userId: string; departmentId: string | null }) => updateUserDepartment(userId, departmentId),
    onSuccess: invalidate,
  });

  return (
    <div className="min-h-screen p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <Link to="/dashboard" className="text-sm text-muted-foreground">← Dashboard</Link>
          <h1 className="text-2xl font-semibold">Team</h1>
        </div>
        {isAdmin && (
          <Button variant="outline" onClick={() => setShowDeptForm((v) => !v)}>
            {showDeptForm ? "Cancel" : "New Department"}
          </Button>
        )}
      </div>

      {showDeptForm && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create department</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="flex gap-3"
              onSubmit={(e) => {
                e.preventDefault();
                deptMutation.mutate();
              }}
            >
              <input
                required
                placeholder="Department name"
                value={deptName}
                onChange={(e) => setDeptName(e.target.value)}
                className="h-10 flex-1 rounded-lg border border-border bg-transparent px-3 text-sm"
              />
              <Button type="submit" disabled={deptMutation.isPending}>
                {deptMutation.isPending ? "Creating…" : "Create"}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {isLoading && <p className="text-muted-foreground">Loading team…</p>}

      {directory && (
        <Card>
          <CardContent className="overflow-x-auto p-0">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border text-left text-xs text-muted-foreground">
                  <th className="p-3">Name</th>
                  <th className="p-3">Email</th>
                  <th className="p-3">Role</th>
                  <th className="p-3">Department</th>
                  <th className="p-3">Status</th>
                  <th className="p-3">Workload</th>
                </tr>
              </thead>
              <tbody>
                {directory.members.map((m: DirectoryEntry) => (
                  <tr key={m.user_id} className="border-b border-border last:border-0">
                    <td className="p-3 font-medium">{m.name}</td>
                    <td className="p-3 text-muted-foreground">{m.email}</td>
                    <td className="p-3">
                      {isAdmin ? (
                        <select
                          value={m.role}
                          onChange={(e) => roleMutation.mutate({ userId: m.user_id, role: e.target.value as OrgRole })}
                          className="h-8 rounded-lg border border-border bg-transparent px-2 text-xs"
                        >
                          {ROLES.map((r) => (
                            <option key={r} value={r}>{r}</option>
                          ))}
                        </select>
                      ) : (
                        <span className="text-xs">{m.role}</span>
                      )}
                    </td>
                    <td className="p-3">
                      {isAdmin ? (
                        <select
                          value={m.department_id ?? ""}
                          onChange={(e) => deptAssignMutation.mutate({ userId: m.user_id, departmentId: e.target.value || null })}
                          className="h-8 rounded-lg border border-border bg-transparent px-2 text-xs"
                        >
                          <option value="">—</option>
                          {directory.departments.map((d) => (
                            <option key={d.id} value={d.id}>{d.name}</option>
                          ))}
                        </select>
                      ) : (
                        <span className="text-xs text-muted-foreground">{m.department_name ?? "—"}</span>
                      )}
                    </td>
                    <td className="p-3">
                      <span className="inline-flex items-center gap-1.5 text-xs">
                        <span className={`h-2 w-2 rounded-full ${m.online ? "bg-primary" : "bg-muted-foreground"}`} />
                        {m.online ? "Online" : "Offline"}
                      </span>
                    </td>
                    <td className="p-3">{m.workload} active</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
