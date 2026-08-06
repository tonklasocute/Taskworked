import { Bell } from "lucide-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listNotifications, markAllRead, markRead } from "@/features/notifications/api";
import { useNotificationSocket } from "@/features/notifications/useNotificationSocket";
import { Button } from "@/components/ui/button";

export default function NotificationBell() {
  useNotificationSocket();
  const queryClient = useQueryClient();

  const { data } = useQuery({ queryKey: ["notifications"], queryFn: listNotifications });
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["notifications"] });

  const readMutation = useMutation({ mutationFn: markRead, onSuccess: invalidate });
  const readAllMutation = useMutation({ mutationFn: markAllRead, onSuccess: invalidate });

  const unread = data?.unread_count ?? 0;

  return (
    // A native <details>/<summary> disclosure needs no open-state or
    // click-outside handling — the browser already does both.
    <details className="group relative">
      <summary className="relative flex h-9 w-9 cursor-pointer list-none items-center justify-center rounded-lg hover:bg-muted [&::-webkit-details-marker]:hidden">
        <Bell className="h-5 w-5" />
        {unread > 0 && (
          <span className="absolute right-1 top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-medium text-destructive-foreground">
            {unread > 9 ? "9+" : unread}
          </span>
        )}
      </summary>

      <div className="glass absolute right-0 z-20 mt-2 w-80 rounded-xl shadow-lg">
        <div className="flex items-center justify-between border-b border-border p-3">
          <span className="text-sm font-semibold">Notifications</span>
          {unread > 0 && (
            <Button size="sm" variant="ghost" onClick={() => readAllMutation.mutate()}>
              Mark all read
            </Button>
          )}
        </div>
        <div className="max-h-96 overflow-y-auto">
          {!data || data.items.length === 0 ? (
            <p className="p-4 text-sm text-muted-foreground">No notifications yet.</p>
          ) : (
            data.items.map((n) => (
              <button
                key={n.id}
                onClick={() => !n.read && readMutation.mutate(n.id)}
                className={`block w-full border-b border-border p-3 text-left text-sm last:border-0 hover:bg-muted ${n.read ? "" : "bg-primary/5"}`}
              >
                <p className="font-medium">{n.title}</p>
                <p className="mt-0.5 text-xs text-muted-foreground">{n.body}</p>
              </button>
            ))
          )}
        </div>
      </div>
    </details>
  );
}
