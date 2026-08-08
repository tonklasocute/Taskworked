import * as React from "react";
import { cn } from "@/lib/utils";

export function DropdownMenu({
  trigger,
  children,
  align = "end",
  className,
}: {
  trigger: React.ReactNode;
  children: React.ReactNode;
  align?: "start" | "end";
  className?: string;
}) {
  return (
    <details className="group relative">
      <summary className="cursor-pointer list-none [&::-webkit-details-marker]:hidden">{trigger}</summary>
      <div
        className={cn(
          "glass absolute z-20 mt-2 min-w-40 rounded-xl p-1 shadow-lg",
          align === "end" ? "right-0" : "left-0",
          className,
        )}
      >
        {children}
      </div>
    </details>
  );
}

export function DropdownMenuItem({ className, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      type="button"
      className={cn("flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm hover:bg-muted", className)}
      {...props}
    />
  );
}
