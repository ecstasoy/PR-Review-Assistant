import { cn } from "@/lib/utils";

// FileStatusBadge GitHub 风格的单字母方块：A/M/D/R + 色块
// status 字符串来自后端 gh.File.Status（added/modified/removed/renamed）
const statusMeta: Record<string, { letter: string; cls: string }> = {
  added: { letter: "A", cls: "bg-ok-bg text-ok" },
  modified: { letter: "M", cls: "bg-info/10 text-info" },
  removed: { letter: "D", cls: "bg-high-bg text-high" },
  renamed: { letter: "R", cls: "bg-med-bg text-med" },
};

export function FileStatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  const m = Object.prototype.hasOwnProperty.call(statusMeta, status)
    ? statusMeta[status]
    : { letter: "?", cls: "bg-surface-2 text-muted" };
  return (
    <span
      title={status}
      className={cn(
        "inline-flex h-4 w-4 shrink-0 items-center justify-center rounded text-[10px] font-semibold",
        m.cls,
        className,
      )}
    >
      {m.letter}
    </span>
  );
}
