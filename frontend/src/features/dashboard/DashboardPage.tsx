import { useAuthStore } from "@/stores/auth-store";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { logout } from "@/features/auth/api";
import { useNavigate, Link } from "react-router-dom";

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const clear = useAuthStore((s) => s.clear);
  const navigate = useNavigate();

  return (
    <div className="min-h-screen p-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Welcome back, {user?.name}</h1>
        <div className="flex gap-2">
          <Link to="/projects">
            <Button variant="outline">Projects</Button>
          </Link>
          <Link to="/team">
            <Button variant="outline">Team</Button>
          </Link>
          <Link to="/gamification">
            <Button variant="outline">Gamification</Button>
          </Link>
          <Link to="/ai-assistant">
            <Button variant="outline">AI Assistant</Button>
          </Link>
          <Button
            variant="outline"
            onClick={async () => {
              await logout();
              clear();
              navigate("/login");
            }}
          >
            Sign out
          </Button>
        </div>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Dashboard</CardTitle>
        </CardHeader>
        <CardContent className="text-muted-foreground">
          Stats, charts, and activity feed land here once the Tasks/Kanban modules exist.
        </CardContent>
      </Card>
    </div>
  );
}
