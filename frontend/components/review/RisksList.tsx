"use client";

import { AlertTriangle } from "lucide-react";

import type { Risk } from "@/lib/types";
import { CategoryBadge, SeverityBadge, type Category } from "@/components/ui/badge";

interface Props {
  risks: Risk[];
}

// 排序键：severity 高 → 低；同 severity 按 confidence 高 → 低
const SEV_RANK: Record<Risk["severity"], number> = { high: 0, medium: 1, low: 2 };

// RisksList 风险识别面板：标题 + 数量；按 severity → confidence 排序
// 每条卡：SeverityBadge + CategoryBadge + file:line + conf NN% + reason
// 对齐 design 原型 ReviewResult 报告视图风险卡布局
export function RisksList({ risks }: Props) {
  if (risks.length === 0) return null;

  const sorted = [...risks].sort((a, b) => {
    const sevDiff = SEV_RANK[a.severity] - SEV_RANK[b.severity];
    if (sevDiff !== 0) return sevDiff;
    return b.confidence - a.confidence;
  });

  return (
    <section className="rounded-lg border border-border bg-surface">
      <header className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2.5">
          <span className="inline-flex h-7 w-7 items-center justify-center rounded-md bg-surface-2 text-accent">
            <AlertTriangle className="h-4 w-4" />
          </span>
          <h2 className="text-sm font-semibold">
            风险识别 <span className="text-muted">{sorted.length} 项</span>
          </h2>
        </div>
        <span className="text-xs text-faint">点击跳到代码（即将上线）</span>
      </header>
      <ul className="divide-y divide-border">
        {sorted.map((r, i) => (
          <RiskRow key={i} risk={r} />
        ))}
      </ul>
    </section>
  );
}

function RiskRow({ risk }: { risk: Risk }) {
  return (
    <li className="px-4 py-3">
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <SeverityBadge severity={risk.severity} />
        <CategoryBadge category={risk.category as Category} />
        <code className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-[11px] text-text-2">
          {risk.file}
          {risk.line ? `:${risk.line}` : ""}
        </code>
        <span className="font-mono text-muted">conf {(risk.confidence * 100).toFixed(0)}%</span>
      </div>
      <p className="mt-2 text-sm leading-relaxed text-text">{risk.reason}</p>
    </li>
  );
}
