export type NotificationType = "assignment" | "comment" | "daily_digest" | "eod_digest" | "weekly_digest" | "monthly_digest";

export interface NotificationItem {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  link?: string;
  read: boolean;
  created_at: string;
}

export interface NotificationList {
  items: NotificationItem[];
  unread_count: number;
}

export interface NotificationPreference {
  email_enabled: boolean;
  line_enabled: boolean;
  line_notify_token?: string;
}
