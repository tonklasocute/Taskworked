import { useAuthStore } from "@/stores/auth-store";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { logout } from "@/features/auth/api";
import { useNavigate } from "react-router-dom";

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const clear = useAuthStore((s) => s.clear);
  const navigate = useNavigate();

  return (
    <div className="min-h-screen p-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Welcome back, {user?.name}</h1>
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
      <Card>
        <CardHeader>
          <CardTitle>Dashboard</CardTitle>
        </CardHeader>
        <CardContent className="text-muted-foreground">
          Stats, charts, and activity feed land here once the Projects/Tasks modules exist.
        </CardContent>
      </Card>
    </div>
  );
}
