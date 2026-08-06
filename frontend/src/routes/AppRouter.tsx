import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import ProtectedRoute from "@/routes/ProtectedRoute";
import LoginPage from "@/features/auth/LoginPage";
import RegisterPage from "@/features/auth/RegisterPage";
import DashboardPage from "@/features/dashboard/DashboardPage";
import ProjectsPage from "@/features/projects/ProjectsPage";
import TaskListPage from "@/features/tasks/TaskListPage";

const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  { path: "/register", element: <RegisterPage /> },
  {
    element: <ProtectedRoute />,
    children: [
      { path: "/dashboard", element: <DashboardPage /> },
      { path: "/projects", element: <ProjectsPage /> },
      { path: "/projects/:projectId", element: <TaskListPage /> },
    ],
  },
  { path: "/", element: <Navigate to="/dashboard" replace /> },
]);

export default function AppRouter() {
  return <RouterProvider router={router} />;
}
