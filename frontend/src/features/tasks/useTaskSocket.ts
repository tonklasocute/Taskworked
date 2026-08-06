import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/auth-store";

// Realtime sync is deliberately dumb: any message for this project just
// invalidates the tasks query and lets TanStack Query refetch. Patching
// the cache in place from the event payload would save a round trip but
// adds a second source of truth to keep in sync — not worth it yet.
export function useTaskSocket(projectId: string | undefined) {
  const queryClient = useQueryClient();
  const accessToken = useAuthStore((s) => s.accessToken);

  useEffect(() => {
    if (!projectId || !accessToken) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${protocol}//${window.location.host}/ws?token=${encodeURIComponent(accessToken)}&project_id=${projectId}`;
    const socket = new WebSocket(url);

    socket.onmessage = () => {
      queryClient.invalidateQueries({ queryKey: ["tasks", projectId] });
    };

    return () => socket.close();
  }, [projectId, accessToken, queryClient]);
}
