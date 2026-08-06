import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/auth-store";

// No project_id — the backend always subscribes every connection to the
// caller's personal "user:<id>" channel regardless, so this is enough to
// receive live notifications from anywhere in the app.
export function useNotificationSocket() {
  const queryClient = useQueryClient();
  const accessToken = useAuthStore((s) => s.accessToken);

  useEffect(() => {
    if (!accessToken) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const url = `${protocol}//${window.location.host}/ws?token=${encodeURIComponent(accessToken)}`;
    const socket = new WebSocket(url);

    socket.onmessage = () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    };

    return () => socket.close();
  }, [accessToken, queryClient]);
}
