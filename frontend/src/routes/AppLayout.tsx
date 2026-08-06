import { Link, Outlet } from "react-router-dom";
import NotificationBell from "@/features/notifications/NotificationBell";

// A slim persistent strip above every protected page — just enough to
// keep the notification bell (and its WebSocket connection) alive no
// matter which page you're on. Each page still owns its own header/nav.
export default function AppLayout() {
  return (
    <div>
      <div className="sticky top-0 z-30 flex items-center justify-end gap-3 border-b border-border bg-background/80 px-4 py-2 backdrop-blur">
        <Link to="/settings/notifications" className="text-xs text-muted-foreground hover:text-foreground">
          Notification settings
        </Link>
        <NotificationBell />
      </div>
      <Outlet />
    </div>
  );
}
