import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { listProjects } from "@/features/projects/api";
import {
  estimateDuration,
  generateMeetingSummary,
  generateTasks,
  getDailySummary,
  getProductivityAnalysis,
  getWeeklySummary,
  predictLateTasks,
  suggestAssignee,
  suggestPriority,
} from "@/features/ai/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

const inputClass = "h-10 w-full rounded-lg border border-border bg-transparent px-3 text-sm";
const textareaClass = "w-full rounded-lg border border-border bg-transparent p-3 text-sm";

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">{children}</CardContent>
    </Card>
  );
}

function ErrorText({ error }: { error: unknown }) {
  if (!error) return null;
  return <p className="text-sm text-destructive">Something went wrong. Please try again.</p>;
}

export default function AIAssistantPage() {
  const { data: projects } = useQuery({ queryKey: ["projects-for-ai"], queryFn: listProjects });
  const [projectId, setProjectId] = useState("");

  const ProjectPicker = (
    <select value={projectId} onChange={(e) => setProjectId(e.target.value)} className={inputClass}>
      <option value="">Select a project…</option>
      {projects?.items.map((p) => (
        <option key={p.id} value={p.id}>
          {p.name}
        </option>
      ))}
    </select>
  );

  // Generate Tasks
  const [genPrompt, setGenPrompt] = useState("");
  const genTasks = useMutation({ mutationFn: () => generateTasks({ project_id: projectId, prompt: genPrompt }) });

  // Daily summary
  const dailySummary = useMutation({ mutationFn: getDailySummary });

  // Weekly summary
  const weeklySummary = useMutation({ mutationFn: () => getWeeklySummary(projectId) });

  // Estimate duration
  const [estTitle, setEstTitle] = useState("");
  const [estDesc, setEstDesc] = useState("");
  const estimate = useMutation({ mutationFn: () => estimateDuration({ title: estTitle, description: estDesc }) });

  // Suggest priority
  const [prTitle, setPrTitle] = useState("");
  const [prDesc, setPrDesc] = useState("");
  const priority = useMutation({ mutationFn: () => suggestPriority({ title: prTitle, description: prDesc }) });

  // Suggest assignee
  const [asTitle, setAsTitle] = useState("");
  const [asDesc, setAsDesc] = useState("");
  const assignee = useMutation({ mutationFn: () => suggestAssignee({ project_id: projectId, title: asTitle, description: asDesc }) });

  // Predict late tasks
  const lateTasks = useMutation({ mutationFn: () => predictLateTasks(projectId) });

  // Productivity analysis
  const productivity = useMutation({ mutationFn: () => getProductivityAnalysis(projectId) });

  // Meeting summary
  const [notes, setNotes] = useState("");
  const meeting = useMutation({ mutationFn: () => generateMeetingSummary({ notes }) });

  return (
    <div className="min-h-screen p-6">
      <div className="mb-6">
        <Link to="/dashboard" className="text-sm text-muted-foreground">
          ← Dashboard
        </Link>
        <h1 className="text-2xl font-semibold">AI Assistant</h1>
        <p className="text-sm text-muted-foreground">Powered by Claude. Suggestions are a starting point — review before acting on them.</p>
      </div>

      <div className="mb-6 max-w-xs">
        <label className="mb-1 block text-xs font-medium text-muted-foreground">Project (for project-scoped tools)</label>
        {ProjectPicker}
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Section title="Generate Tasks">
          <textarea
            className={textareaClass}
            rows={3}
            placeholder="Describe a goal or feature, e.g. 'Add password reset via email'"
            value={genPrompt}
            onChange={(e) => setGenPrompt(e.target.value)}
          />
          <Button size="sm" disabled={!projectId || !genPrompt || genTasks.isPending} onClick={() => genTasks.mutate()}>
            {genTasks.isPending ? "Generating…" : "Generate tasks"}
          </Button>
          <ErrorText error={genTasks.error} />
          {genTasks.data && (
            <ul className="flex flex-col gap-2">
              {genTasks.data.tasks.map((t, i) => (
                <li key={i} className="rounded-lg border border-border p-3 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="font-medium">{t.title}</span>
                    <span className="text-xs text-muted-foreground">{t.priority} · {t.estimate_hours}h</span>
                  </div>
                  <p className="mt-1 text-muted-foreground">{t.description}</p>
                </li>
              ))}
            </ul>
          )}
        </Section>

        <Section title="Daily Summary">
          <p className="text-sm text-muted-foreground">A quick digest of your active tasks for today.</p>
          <Button size="sm" disabled={dailySummary.isPending} onClick={() => dailySummary.mutate()}>
            {dailySummary.isPending ? "Summarizing…" : "Summarize my day"}
          </Button>
          <ErrorText error={dailySummary.error} />
          {dailySummary.data && <p className="text-sm">{dailySummary.data.summary}</p>}
        </Section>

        <Section title="Weekly Report Summary">
          <p className="text-sm text-muted-foreground">Narrative summary of the selected project's last 7 days.</p>
          <Button size="sm" disabled={!projectId || weeklySummary.isPending} onClick={() => weeklySummary.mutate()}>
            {weeklySummary.isPending ? "Summarizing…" : "Summarize this week"}
          </Button>
          <ErrorText error={weeklySummary.error} />
          {weeklySummary.data && <p className="text-sm">{weeklySummary.data.summary}</p>}
        </Section>

        <Section title="Team Productivity Analysis">
          <p className="text-sm text-muted-foreground">Analysis of the selected project's completion trends over the last 30 days.</p>
          <Button size="sm" disabled={!projectId || productivity.isPending} onClick={() => productivity.mutate()}>
            {productivity.isPending ? "Analyzing…" : "Analyze productivity"}
          </Button>
          <ErrorText error={productivity.error} />
          {productivity.data && <p className="text-sm">{productivity.data.summary}</p>}
        </Section>

        <Section title="Predict Late Tasks">
          <p className="text-sm text-muted-foreground">Flags open tasks in the selected project that look at risk of finishing late.</p>
          <Button size="sm" disabled={!projectId || lateTasks.isPending} onClick={() => lateTasks.mutate()}>
            {lateTasks.isPending ? "Checking…" : "Predict late tasks"}
          </Button>
          <ErrorText error={lateTasks.error} />
          {lateTasks.data && (
            <ul className="flex flex-col gap-2">
              {lateTasks.data.at_risk.length === 0 && <li className="text-sm text-muted-foreground">No tasks currently look at risk.</li>}
              {lateTasks.data.at_risk.map((r) => (
                <li key={r.task_id} className="rounded-lg border border-border p-3 text-sm">
                  <span className="font-medium">{r.title}</span>
                  <p className="mt-1 text-muted-foreground">{r.reasoning}</p>
                </li>
              ))}
            </ul>
          )}
        </Section>

        <Section title="Estimate Duration">
          <input className={inputClass} placeholder="Task title" value={estTitle} onChange={(e) => setEstTitle(e.target.value)} />
          <textarea className={textareaClass} rows={2} placeholder="Description" value={estDesc} onChange={(e) => setEstDesc(e.target.value)} />
          <Button size="sm" disabled={!estTitle || estimate.isPending} onClick={() => estimate.mutate()}>
            {estimate.isPending ? "Estimating…" : "Estimate hours"}
          </Button>
          <ErrorText error={estimate.error} />
          {estimate.data && (
            <p className="text-sm">
              <span className="font-medium">{estimate.data.estimate_hours}h</span> — {estimate.data.reasoning}
            </p>
          )}
        </Section>

        <Section title="Suggest Priority">
          <input className={inputClass} placeholder="Task title" value={prTitle} onChange={(e) => setPrTitle(e.target.value)} />
          <textarea className={textareaClass} rows={2} placeholder="Description" value={prDesc} onChange={(e) => setPrDesc(e.target.value)} />
          <Button size="sm" disabled={!prTitle || priority.isPending} onClick={() => priority.mutate()}>
            {priority.isPending ? "Thinking…" : "Suggest priority"}
          </Button>
          <ErrorText error={priority.error} />
          {priority.data && (
            <p className="text-sm">
              <span className="font-medium capitalize">{priority.data.priority}</span> — {priority.data.reasoning}
            </p>
          )}
        </Section>

        <Section title="Suggest Assignee">
          <input className={inputClass} placeholder="Task title" value={asTitle} onChange={(e) => setAsTitle(e.target.value)} />
          <textarea className={textareaClass} rows={2} placeholder="Description" value={asDesc} onChange={(e) => setAsDesc(e.target.value)} />
          <Button size="sm" disabled={!projectId || !asTitle || assignee.isPending} onClick={() => assignee.mutate()}>
            {assignee.isPending ? "Thinking…" : "Suggest assignee"}
          </Button>
          <ErrorText error={assignee.error} />
          {assignee.data && (
            <p className="text-sm">
              <span className="font-medium">{assignee.data.assignee_name}</span> — {assignee.data.reasoning}
            </p>
          )}
        </Section>

        <Section title="Meeting Summary">
          <textarea
            className={textareaClass}
            rows={5}
            placeholder="Paste raw meeting notes here…"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
          />
          <Button size="sm" disabled={!notes || meeting.isPending} onClick={() => meeting.mutate()}>
            {meeting.isPending ? "Summarizing…" : "Summarize meeting"}
          </Button>
          <ErrorText error={meeting.error} />
          {meeting.data && (
            <div className="text-sm">
              <p>{meeting.data.summary}</p>
              {meeting.data.action_items.length > 0 && (
                <ul className="mt-2 list-disc pl-5">
                  {meeting.data.action_items.map((item, i) => (
                    <li key={i}>{item}</li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </Section>
      </div>
    </div>
  );
}
