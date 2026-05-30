import { Check } from "lucide-react";

import { cn } from "@/lib/utils";
import { Spinner } from "@/components/ui/spinner";

export type StageState = "pending" | "active" | "done";

interface Props {
  label: string;
  state: StageState;
  className?: string;
}

// StageChip 顶栏阶段状态徽标（总结 / 风险 / 建议）。
// pending: 空心圆 + faint 文字；active: Spinner + accent 文字；done: ✓ ok + text-2 文字。
// 完全对齐 design 原型 ReviewResult.jsx 的 StageChip。
export function StageChip({ label, state, className }: Props) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 text-xs font-medium",
        state === "done" ? "text-text-2" : state === "active" ? "text-accent" : "text-faint",
        className,
      )}
    >
      {state === "done" ? (
        <Check className="h-3 w-3 text-ok" strokeWidth={2.4} />
      ) : state === "active" ? (
        <Spinner size="xs" className="text-accent" />
      ) : (
        <span className="inline-block h-3 w-3 rounded-full border-[1.5px] border-border-strong" />
      )}
      {label}
    </span>
  );
}
