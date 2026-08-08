import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { assignTaskToMilestone, createGoal, createMilestone, getActionPlan } from "@/features/actionplan/api";
import type { GoalNode, MilestoneNode } from "@/features/actionplan/types";
import { listTasks } from "@/features/tasks/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Badge, type BadgeProps } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";

const STATUS_LABEL: Record<string, string> = {
  not_started: "Not started",
  in_progress: "In progress",
  completed: "Completed",
};

const STATUS_VARIANT: Record<string, BadgeProps["variant"]> = {
  not_started: "default",
  in_progress: "primary",
  completed: "outline",
};

export default function ActionPlanPage() {
  const { projectId } = useParams<{ projectId: string }>();
  const queryClient = useQueryClient();

  const [showGoalForm, setShowGoalForm] = useState(false);
  const [goalName, setGoalName] = useState("");
  const [milestoneFormGoalId, setMilestoneFormGoalId] = useState<string | null>(null);
  const [milestoneName, setMilestoneName] = useState("");
  const [assignFormMilestoneId, setAssignFormMilestoneId] = useState<string | null>(null);
  const [assignTaskId, setAssignTaskId] = useState("");

  const { data: plan, isLoading } = useQuery({
    queryKey: ["actionplan", projectId],
    queryFn: () => getActionPlan(projectId!),
    enabled: !!projectId,
  });

  const { data: allTasks } = useQuery({
    queryKey: ["tasks", projectId],
    queryFn: () => listTasks(projectId!),
    enabled: !!projectId,
  });
  const unplannedTasks = (allTasks?.items ?? []).filter((t) => !t.milestone_id && !t.parent_task_id);

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["actionplan", projectId] });
    queryClient.invalidateQueries({ queryKey: ["tasks", projectId] });
  };

  const goalMutation = useMutation({
    mutationFn: () => createGoal({ project_id: projectId!, name: goalName }),
    onSuccess: () => {
      invalidate();
      setGoalName("");
      setShowGoalForm(false);
    },
  });

  const milestoneMutation = useMutation({
    mutationFn: (goalId: string) => createMilestone({ goal_id: goalId, name: milestoneName }),
    onSuccess: () => {
      invalidate();
      setMilestoneName("");
      setMilestoneFormGoalId(null);
    },
  });

  const assignMutation = useMutation({
    mutationFn: ({ taskId, milestoneId }: { taskId: string; milestoneId: string }) => assignTaskToMilestone(taskId, milestoneId),
    onSuccess: () => {
      invalidate();
      setAssignTaskId("");
      setAssignFormMilestoneId(null);
    },
  });

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-end">
        <Button onClick={() => setShowGoalForm((v) => !v)}>{showGoalForm ? "Cancel" : "New Goal"}</Button>
      </div>

      {showGoalForm && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create goal</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="flex gap-3"
              onSubmit={(e) => {
                e.preventDefault();
                goalMutation.mutate();
              }}
            >
              <Input required placeholder="Goal name" value={goalName} onChange={(e) => setGoalName(e.target.value)} className="flex-1" />
              <Button type="submit" disabled={goalMutation.isPending}>
                {goalMutation.isPending ? "Creating…" : "Create"}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {!isLoading && plan?.goals.length === 0 && (
        <EmptyState title="No goals yet" description="Create one to start structuring this project's roadmap." />
      )}

      <div className="flex flex-col gap-4">
        {plan?.goals.map((goal: GoalNode) => (
          <Card key={goal.id}>
            <CardHeader className="flex flex-row items-center justify-between">
              <div>
                <CardTitle>{goal.name}</CardTitle>
                <Badge variant={STATUS_VARIANT[goal.status]} className="mt-1">
                  {STATUS_LABEL[goal.status]}
                </Badge>
              </div>
              <Button size="sm" variant="outline" onClick={() => setMilestoneFormGoalId(milestoneFormGoalId === goal.id ? null : goal.id)}>
                {milestoneFormGoalId === goal.id ? "Cancel" : "+ Milestone"}
              </Button>
            </CardHeader>
            <CardContent>
              {milestoneFormGoalId === goal.id && (
                <form
                  className="mb-4 flex gap-2"
                  onSubmit={(e) => {
                    e.preventDefault();
                    milestoneMutation.mutate(goal.id);
                  }}
                >
                  <Input
                    required
                    placeholder="Milestone name"
                    value={milestoneName}
                    onChange={(e) => setMilestoneName(e.target.value)}
                    className="h-9 flex-1"
                  />
                  <Button type="submit" size="sm" disabled={milestoneMutation.isPending}>
                    Add
                  </Button>
                </form>
              )}

              {goal.milestones.length === 0 ? (
                <p className="text-sm text-muted-foreground">No milestones yet.</p>
              ) : (
                <div className="flex flex-col gap-3">
                  {goal.milestones.map((milestone: MilestoneNode) => (
                    <div key={milestone.id} className="rounded-lg border border-border p-3">
                      <div className="mb-2 flex items-center justify-between">
                        <div>
                          <p className="text-sm font-medium">{milestone.name}</p>
                          <Badge variant={STATUS_VARIANT[milestone.status]} className="mt-1">
                            {STATUS_LABEL[milestone.status]}
                          </Badge>
                        </div>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => setAssignFormMilestoneId(assignFormMilestoneId === milestone.id ? null : milestone.id)}
                        >
                          {assignFormMilestoneId === milestone.id ? "Cancel" : "+ Task"}
                        </Button>
                      </div>

                      {assignFormMilestoneId === milestone.id && (
                        <form
                          className="mb-3 flex gap-2"
                          onSubmit={(e) => {
                            e.preventDefault();
                            if (assignTaskId) assignMutation.mutate({ taskId: assignTaskId, milestoneId: milestone.id });
                          }}
                        >
                          <Select value={assignTaskId} onChange={(e) => setAssignTaskId(e.target.value)} className="h-9 flex-1">
                            <option value="">Select an unplanned task…</option>
                            {unplannedTasks.map((t) => (
                              <option key={t.id} value={t.id}>{t.title}</option>
                            ))}
                          </Select>
                          <Button type="submit" size="sm" disabled={!assignTaskId || assignMutation.isPending}>
                            Assign
                          </Button>
                        </form>
                      )}

                      {milestone.tasks.length === 0 ? (
                        <p className="text-xs text-muted-foreground">No tasks assigned.</p>
                      ) : (
                        <ul className="flex flex-col gap-1">
                          {milestone.tasks.map((task) => (
                            <li key={task.id} className="text-sm">
                              <span>{task.title}</span>
                              {task.subtasks.length > 0 && (
                                <ul className="ml-4 mt-1 flex flex-col gap-0.5 border-l border-border pl-3">
                                  {task.subtasks.map((sub) => (
                                    <li key={sub.id} className="text-xs text-muted-foreground">{sub.title}</li>
                                  ))}
                                </ul>
                              )}
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
