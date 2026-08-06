import { api } from "@/lib/api";
import type { PeriodReport } from "@/features/reports/types";

export function getPeriodReport(projectId: string, period: "week" | "month") {
  return api.get<{ data: PeriodReport }>(`/projects/${projectId}/reports`, { params: { period } }).then((r) => r.data.data);
}

// A plain <a href> download wouldn't carry the Authorization header, so
// the file is fetched through the authenticated api client and turned
// into a Blob URL instead.
export async function downloadCsv(projectId: string) {
  const res = await api.get(`/projects/${projectId}/reports/export.csv`, { responseType: "blob" });
  const url = window.URL.createObjectURL(new Blob([res.data]));
  const link = document.createElement("a");
  link.href = url;
  link.download = "tasks.csv";
  link.click();
  window.URL.revokeObjectURL(url);
}
