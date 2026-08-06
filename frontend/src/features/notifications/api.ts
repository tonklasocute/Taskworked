import { api } from "@/lib/api";
import type { NotificationList, NotificationPreference } from "@/features/notifications/types";

export function listNotifications() {
  return api.get<{ data: NotificationList }>("/notifications").then((r) => r.data.data);
}

export function markRead(id: string) {
  return api.post(`/notifications/${id}/read`);
}

export function markAllRead() {
  return api.post("/notifications/read-all");
}

export function getPreference() {
  return api.get<{ data: NotificationPreference }>("/notification-preferences").then((r) => r.data.data);
}

export function updatePreference(input: Partial<NotificationPreference>) {
  return api.patch<{ data: NotificationPreference }>("/notification-preferences", input).then((r) => r.data.data);
}
