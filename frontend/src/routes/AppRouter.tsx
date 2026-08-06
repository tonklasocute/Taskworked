import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import ProtectedRoute from "@/routes/ProtectedRoute";
import LoginPage from "@/features/auth/LoginPage";
import RegisterPage from "@/features/auth/RegisterPage";
import DashboardPage from "@/features/dashboard/DashboardPage";
import ProjectsPage from "@/features/projects/ProjectsPage";
import TaskListPage from "@/features/tasks/TaskListPage";
import KanbanBoard from "@/features/tasks/KanbanBoard";
import CalendarPage from "@/features/tasks/CalendarPage";
import GanttPage from "@/features/tasks/GanttPage";
import ActionPlanPage from "@/features/actionplan/ActionPlanPage";
import ReportsPage from "@/features/reports/ReportsPage";
import TeamPage from "@/features/team/TeamPage";

const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  { path: "/register", element: <RegisterPage /> },
  {
    element: <ProtectedRoute />,
    children: [
      { path: "/dashboard", element: <DashboardPage /> },
      { path: "/projects", element: <ProjectsPage /> },
      { path: "/projects/:projectId", element: <TaskListPage /> },
      { path: "/projects/:projectId/board", element: <KanbanBoard /> },
      { path: "/projects/:projectId/calendar", element: <CalendarPage /> },
      { path: "/projects/:projectId/gantt", element: <GanttPage /> },
      { path: "/projects/:projectId/action-plan", element: <ActionPlanPage /> },
      { path: "/projects/:projectId/reports", element: <ReportsPage /> },
      { path: "/team", element: <TeamPage /> },
    ],
  },
  { path: "/", element: <Navigate to="/dashboard" replace /> },
]);

export default function AppRouter() {
  return <RouterProvider router={router} />;
}
