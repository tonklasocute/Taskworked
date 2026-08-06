import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { createProject, listProjects } from "@/features/projects/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

const STATUS_LABEL: Record<string, string> = {
  planning: "Planning",
  active: "Active",
  on_hold: "On Hold",
  completed: "Completed",
  archived: "Archived",
};

export default function ProjectsPage() {
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");

  const { data, isLoading } = useQuery({ queryKey: ["projects"], queryFn: listProjects });

  const mutation = useMutation({
    mutationFn: () => createProject({ name, description }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
      setName("");
      setDescription("");
      setShowForm(false);
    },
  });

  return (
    <div className="min-h-screen p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <Link to="/dashboard" className="text-sm text-muted-foreground">← Dashboard</Link>
          <h1 className="text-2xl font-semibold">Projects</h1>
        </div>
        <Button onClick={() => setShowForm((v) => !v)}>{showForm ? "Cancel" : "New Project"}</Button>
      </div>

      {showForm && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create project</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="flex flex-col gap-3"
              onSubmit={(e) => {
                e.preventDefault();
                mutation.mutate();
              }}
            >
              <input
                required
                placeholder="Project name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="h-10 rounded-lg border border-border bg-transparent px-3 text-sm"
              />
              <textarea
                placeholder="Description (optional)"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="min-h-20 rounded-lg border border-border bg-transparent px-3 py-2 text-sm"
              />
              <Button type="submit" disabled={mutation.isPending} className="self-start">
                {mutation.isPending ? "Creating…" : "Create project"}
              </Button>
            </form>
          </CardContent>
        </Card>
      )}

      {isLoading && <p className="text-muted-foreground">Loading projects…</p>}

      {!isLoading && data?.items.length === 0 && (
        <p className="text-muted-foreground">No projects yet. Create your first one above.</p>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {data?.items.map((project) => (
          <Link key={project.id} to={`/projects/${project.id}`}>
            <Card className="h-full transition-opacity hover:opacity-90">
              <CardHeader>
                <CardTitle>{project.name}</CardTitle>
              </CardHeader>
              <CardContent>
                {project.description && (
                  <p className="mb-3 line-clamp-2 text-sm text-muted-foreground">{project.description}</p>
                )}
                <span className="inline-flex rounded-full bg-muted px-2 py-1 text-xs font-medium text-muted-foreground">
                  {STATUS_LABEL[project.status] ?? project.status}
                </span>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}
