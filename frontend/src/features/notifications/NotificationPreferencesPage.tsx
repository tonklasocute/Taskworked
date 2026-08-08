import { useEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { getPreference, updatePreference } from "@/features/notifications/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export default function NotificationPreferencesPage() {
  const { data } = useQuery({ queryKey: ["notification-preference"], queryFn: getPreference });

  const [emailEnabled, setEmailEnabled] = useState(true);
  const [lineEnabled, setLineEnabled] = useState(false);
  const [lineToken, setLineToken] = useState("");

  useEffect(() => {
    if (!data) return;
    setEmailEnabled(data.email_enabled);
    setLineEnabled(data.line_enabled);
    setLineToken(data.line_notify_token ?? "");
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: () => updatePreference({ email_enabled: emailEnabled, line_enabled: lineEnabled, line_notify_token: lineToken }),
  });

  return (
    <div className="p-6">
      <h1 className="mb-6 text-2xl font-semibold">Notification Settings</h1>

      <Card className="max-w-md">
        <CardHeader>
          <CardTitle>Delivery channels</CardTitle>
        </CardHeader>
        <CardContent>
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault();
              saveMutation.mutate();
            }}
          >
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={emailEnabled} onChange={(e) => setEmailEnabled(e.target.checked)} />
              Email notifications
            </label>

            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={lineEnabled} onChange={(e) => setLineEnabled(e.target.checked)} />
              LINE Notify
            </label>

            {lineEnabled && (
              <div className="flex flex-col gap-1">
                <Input placeholder="LINE Notify token" value={lineToken} onChange={(e) => setLineToken(e.target.value)} />
                <p className="text-xs text-muted-foreground">
                  Get a personal token from{" "}
                  <a href="https://notify-bot.line.me/my/" target="_blank" rel="noreferrer" className="underline">
                    notify-bot.line.me
                  </a>
                </p>
              </div>
            )}

            <Button type="submit" disabled={saveMutation.isPending} className="self-start">
              {saveMutation.isPending ? "Saving…" : "Save"}
            </Button>
            {saveMutation.isSuccess && <p className="text-sm text-primary">Saved.</p>}
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
