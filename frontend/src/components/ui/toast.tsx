import { AnimatePresence, motion } from "framer-motion";
import { CheckCircle2, Info, X, XCircle } from "lucide-react";
import { useToastStore, type Toast, type ToastVariant } from "@/stores/toast-store";
import { cn } from "@/lib/utils";

const ICON: Record<ToastVariant, typeof CheckCircle2> = {
  success: CheckCircle2,
  error: XCircle,
  info: Info,
};

const COLOR: Record<ToastVariant, string> = {
  success: "text-primary",
  error: "text-destructive",
  info: "text-foreground",
};

function ToastItem({ toast }: { toast: Toast }) {
  const dismiss = useToastStore((s) => s.dismiss);
  const Icon = ICON[toast.variant];

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: -8, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      className="glass flex w-80 items-start gap-2 rounded-xl p-3 shadow-lg"
    >
      <Icon className={cn("mt-0.5 h-4 w-4 shrink-0", COLOR[toast.variant])} />
      <p className="flex-1 text-sm">{toast.message}</p>
      <button onClick={() => dismiss(toast.id)} className="text-muted-foreground hover:text-foreground" aria-label="Dismiss">
        <X className="h-4 w-4" />
      </button>
    </motion.div>
  );
}

export function Toaster() {
  const toasts = useToastStore((s) => s.toasts);

  return (
    <div className="pointer-events-none fixed right-4 top-4 z-100 flex flex-col gap-2">
      <AnimatePresence>
        {toasts.map((t) => (
          <div key={t.id} className="pointer-events-auto">
            <ToastItem toast={t} />
          </div>
        ))}
      </AnimatePresence>
    </div>
  );
}
