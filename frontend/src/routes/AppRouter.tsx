import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import ProtectedRoute from "@/routes/ProtectedRoute";
import AppShell from "@/routes/AppLayout";
import ProjectLayout from "@/routes/ProjectLayout";
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
import NotificationPreferencesPage from "@/features/notifications/NotificationPreferencesPage";
import GamificationPage from "@/features/gamification/GamificationPage";
import AIAssistantPage from "@/features/ai/AIAssistantPage";

const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  { path: "/register", element: <RegisterPage /> },
  {
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppShell />,
        children: [
          { path: "/dashboard", element: <DashboardPage /> },
          { path: "/projects", element: <ProjectsPage /> },
          {
            path: "/projects/:projectId",
            element: <ProjectLayout />,
            children: [
              { index: true, element: <TaskListPage /> },
              { path: "board", element: <KanbanBoard /> },
              { path: "calendar", element: <CalendarPage /> },
              { path: "gantt", element: <GanttPage /> },
              { path: "action-plan", element: <ActionPlanPage /> },
              { path: "reports", element: <ReportsPage /> },
            ],
          },
          { path: "/team", element: <TeamPage /> },
          { path: "/settings/notifications", element: <NotificationPreferencesPage /> },
          { path: "/gamification", element: <GamificationPage /> },
          { path: "/ai-assistant", element: <AIAssistantPage /> },
        ],
      },
    ],
  },
  { path: "/", element: <Navigate to="/dashboard" replace /> },
]);

export default function AppRouter() {
  return <RouterProvider router={router} />;
}
