import { cn } from "@/lib/utils";

export type CIStatusValue = "passing" | "failing" | "pending";

const dotClass: Record<CIStatusValue, string> = {
  passing: "bg-ok",
  failing: "bg-fail",
  pending: "bg-pending animate-pulse-dot",
};

const labelText: Record<CIStatusValue, string> = {
  passing: "通过",
  failing: "失败",
  pending: "进行中",
};

// CIStatus 圆点（默认 8px）；可选附 label。pending 自带 pulse-dot 呼吸动画。
// 状态值非法或空时不渲染（让调用方决定 fallback，避免渲染"未知"状态误导）。
export function CIStatus({
  status,
  withLabel = false,
  className,
}: {
  status: CIStatusValue | string;
  withLabel?: boolean;
  className?: string;
}) {
  if (status !== "passing" && status !== "failing" && status !== "pending") {
    return null;
  }
  return (
    <span
      className={cn("inline-flex items-center gap-1.5", className)}
      aria-label={`CI: ${labelText[status]}`}
    >
      <span className={cn("h-2 w-2 rounded-full", dotClass[status])} aria-hidden />
      {withLabel ? (
        <span className="text-xs text-muted">{labelText[status]}</span>
      ) : null}
    </span>
  );
}
