import { Link, NavLink, Outlet, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { getProject } from "@/features/projects/api";
import { cn } from "@/lib/utils";

const TABS = [
  { to: "", label: "List", end: true },
  { to: "board", label: "Board", end: false },
  { to: "calendar", label: "Calendar", end: false },
  { to: "gantt", label: "Gantt", end: false },
  { to: "action-plan", label: "Action Plan", end: false },
  { to: "reports", label: "Reports", end: false },
];

export default function ProjectLayout() {
  const { projectId } = useParams<{ projectId: string }>();

  const { data: project } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => getProject(projectId!),
    enabled: !!projectId,
  });

  return (
    <div className="flex min-h-screen flex-col">
      <div className="border-b border-border px-6 pt-6 print:hidden">
        <Link to="/projects" className="mb-2 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
          <ArrowLeft className="h-4 w-4" /> Projects
        </Link>
        <h1 className="mb-4 text-2xl font-semibold">{project?.name ?? "Project"}</h1>
        <nav className="flex gap-1">
          {TABS.map((tab) => (
            <NavLink
              key={tab.label}
              to={tab.to}
              end={tab.end}
              className={({ isActive }) =>
                cn(
                  "rounded-t-lg px-3 py-2 text-sm font-medium",
                  isActive ? "border-b-2 border-primary text-primary" : "text-muted-foreground hover:text-foreground",
                )
              }
            >
              {tab.label}
            </NavLink>
          ))}
        </nav>
      </div>
      <div className="flex-1">
        <Outlet />
      </div>
    </div>
  );
}
