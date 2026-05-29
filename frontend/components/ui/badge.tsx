import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

// Badge 通用小标签，类似 shadcn/ui Badge。
// SeverityBadge / CategoryBadge 是上面再包一层的语义化导出，把 Risk/Suggestion 的枚举值映射到色彩 token。
const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-sm border px-2 py-0.5 text-xs font-medium",
  {
    variants: {
      tone: {
        neutral: "border-border bg-surface-2 text-text-2",
        accent: "border-transparent bg-accent text-accent-fg",
        outline: "border-border-strong bg-transparent text-text",
      },
    },
    defaultVariants: { tone: "neutral" },
  },
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />;
}

// ── 风险等级徽章 ──────────────────────────────────────────────
export type Severity = "high" | "medium" | "low";

const severityClass: Record<Severity, string> = {
  high: "border-high-bd bg-high-bg text-high",
  medium: "border-med-bd bg-med-bg text-med",
  low: "border-low-bd bg-low-bg text-low",
};

const severityLabel: Record<Severity, string> = {
  high: "high",
  medium: "medium",
  low: "low",
};

export function SeverityBadge({
  severity,
  className,
  children,
}: {
  severity: Severity;
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-sm border px-2 py-0.5 text-xs font-medium uppercase",
        severityClass[severity],
        className,
      )}
    >
      {children ?? severityLabel[severity]}
    </span>
  );
}

// ── 类别徽章 ──────────────────────────────────────────────
// bug / security / concurrency 高严重度（红）
// perf 中（amber）
// style / other 低（zinc）
export type Category = "bug" | "security" | "concurrency" | "perf" | "style" | "other";

const categoryClass: Record<Category, string> = {
  bug: "border-high-bd bg-high-bg text-high",
  security: "border-high-bd bg-high-bg text-high",
  concurrency: "border-high-bd bg-high-bg text-high",
  perf: "border-med-bd bg-med-bg text-med",
  style: "border-low-bd bg-low-bg text-low",
  other: "border-border bg-surface-2 text-muted",
};

export function CategoryBadge({
  category,
  className,
}: {
  category: Category | string; // 容忍未知值 → 走 other
  className?: string;
}) {
  const c = (
    Object.prototype.hasOwnProperty.call(categoryClass, category) ? category : "other"
  ) as Category;
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-sm border px-2 py-0.5 text-xs font-medium",
        categoryClass[c],
        className,
      )}
    >
      {category}
    </span>
  );
}
