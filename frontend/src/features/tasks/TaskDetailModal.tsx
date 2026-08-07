import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/auth-store";
import {
  addComment,
  deleteAttachment,
  deleteComment,
  getAttachmentDownloadURL,
  getTask,
  getWatchStatus,
  listAttachments,
  listComments,
  listWatchers,
  unwatchTask,
  uploadAttachment,
  watchTask,
} from "@/features/tasks/api";
import { Button } from "@/components/ui/button";

const inputClass = "h-10 flex-1 rounded-lg border border-border bg-transparent px-3 text-sm";

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function TaskDetailModal({ taskId, onClose }: { taskId: string; onClose: () => void }) {
  const queryClient = useQueryClient();
  const currentUserId = useAuthStore((s) => s.user?.id);
  const [commentBody, setCommentBody] = useState("");

  const { data: task } = useQuery({ queryKey: ["task", taskId], queryFn: () => getTask(taskId) });
  const { data: comments } = useQuery({ queryKey: ["task-comments", taskId], queryFn: () => listComments(taskId) });
  const { data: attachments } = useQuery({ queryKey: ["task-attachments", taskId], queryFn: () => listAttachments(taskId) });
  const { data: watchers } = useQuery({ queryKey: ["task-watchers", taskId], queryFn: () => listWatchers(taskId) });
  const { data: watchStatus } = useQuery({ queryKey: ["task-watch-status", taskId], queryFn: () => getWatchStatus(taskId) });

  const invalidateComments = () => queryClient.invalidateQueries({ queryKey: ["task-comments", taskId] });
  const addCommentMutation = useMutation({
    mutationFn: () => addComment(taskId, commentBody),
    onSuccess: () => {
      setCommentBody("");
      invalidateComments();
    },
  });
  const deleteCommentMutation = useMutation({ mutationFn: (id: string) => deleteComment(taskId, id), onSuccess: invalidateComments });

  const invalidateAttachments = () => queryClient.invalidateQueries({ queryKey: ["task-attachments", taskId] });
  const uploadMutation = useMutation({ mutationFn: (file: File) => uploadAttachment(taskId, file), onSuccess: invalidateAttachments });
  const deleteAttachmentMutation = useMutation({
    mutationFn: (id: string) => deleteAttachment(taskId, id),
    onSuccess: invalidateAttachments,
  });
  const downloadMutation = useMutation({
    mutationFn: (id: string) => getAttachmentDownloadURL(taskId, id),
    onSuccess: (url) => window.open(url, "_blank"),
  });

  const invalidateWatch = () => {
    queryClient.invalidateQueries({ queryKey: ["task-watchers", taskId] });
    queryClient.invalidateQueries({ queryKey: ["task-watch-status", taskId] });
  };
  const watchMutation = useMutation({ mutationFn: () => watchTask(taskId), onSuccess: invalidateWatch });
  const unwatchMutation = useMutation({ mutationFn: () => unwatchTask(taskId), onSuccess: invalidateWatch });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div
        className="glass max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold">{task?.title ?? "Loading…"}</h2>
            {task?.description && <p className="mt-1 text-sm text-muted-foreground">{task.description}</p>}
          </div>
          <Button size="sm" variant="ghost" onClick={onClose}>
            ✕
          </Button>
        </div>

        <div className="mb-6 flex items-center gap-3">
          <Button
            size="sm"
            variant={watchStatus?.watching ? "default" : "outline"}
            onClick={() => (watchStatus?.watching ? unwatchMutation.mutate() : watchMutation.mutate())}
          >
            {watchStatus?.watching ? "★ Watching" : "☆ Watch"}
          </Button>
          <span className="text-xs text-muted-foreground">
            {watchers?.length ?? 0} watching
            {watchers && watchers.length > 0 ? `: ${watchers.map((w) => w.name).join(", ")}` : ""}
          </span>
        </div>

        <section className="mb-6">
          <h3 className="mb-2 text-sm font-semibold">Attachments</h3>
          <ul className="mb-3 flex flex-col gap-2">
            {attachments?.map((a) => (
              <li key={a.id} className="flex items-center justify-between rounded-lg border border-border p-2 text-sm">
                <button className="text-left hover:underline" onClick={() => downloadMutation.mutate(a.id)}>
                  {a.file_name} <span className="text-xs text-muted-foreground">({formatSize(a.size_bytes)})</span>
                </button>
                {a.uploader_id === currentUserId && (
                  <Button size="sm" variant="ghost" onClick={() => deleteAttachmentMutation.mutate(a.id)}>
                    Delete
                  </Button>
                )}
              </li>
            ))}
            {attachments?.length === 0 && <p className="text-sm text-muted-foreground">No attachments yet.</p>}
          </ul>
          <input
            type="file"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) uploadMutation.mutate(file);
              e.target.value = "";
            }}
            disabled={uploadMutation.isPending}
            className="text-sm"
          />
          {uploadMutation.isError && <p className="mt-1 text-xs text-destructive">Upload failed. Is object storage configured?</p>}
        </section>

        <section>
          <h3 className="mb-2 text-sm font-semibold">Comments</h3>
          <ul className="mb-3 flex flex-col gap-3">
            {comments?.map((c) => (
              <li key={c.id} className="rounded-lg border border-border p-3 text-sm">
                <div className="mb-1 flex items-center justify-between">
                  <span className="font-medium">{c.author_name}</span>
                  <span className="text-xs text-muted-foreground">{new Date(c.created_at).toLocaleString()}</span>
                </div>
                <p className="whitespace-pre-wrap">{c.body}</p>
                {c.author_id === currentUserId && (
                  <button
                    className="mt-1 text-xs text-muted-foreground hover:text-destructive"
                    onClick={() => deleteCommentMutation.mutate(c.id)}
                  >
                    Delete
                  </button>
                )}
              </li>
            ))}
            {comments?.length === 0 && <p className="text-sm text-muted-foreground">No comments yet.</p>}
          </ul>
          <form
            className="flex gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              if (commentBody.trim()) addCommentMutation.mutate();
            }}
          >
            <input
              className={inputClass}
              placeholder="Write a comment…"
              value={commentBody}
              onChange={(e) => setCommentBody(e.target.value)}
            />
            <Button type="submit" size="sm" disabled={addCommentMutation.isPending}>
              Post
            </Button>
          </form>
        </section>
      </div>
    </div>
  );
}
