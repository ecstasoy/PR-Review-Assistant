import type { Risk } from "@/lib/types";

interface Props {
  risks: Risk[];
}

const severityRank = { high: 0, medium: 1, low: 2 } as const;

const severityClass = {
  high: "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300",
  medium: "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300",
  low: "bg-zinc-100 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-400",
} as const;

// RiskList 按 severity（high → low）+ confidence（高 → 低）双重排序
// confidence ≥ 0.9 默认展开，其它折叠到 <details> 里
export function RiskList({ risks }: Props) {
  if (risks.length === 0) return null;

  const sorted = [...risks].sort((a, b) => {
    if (severityRank[a.severity] !== severityRank[b.severity]) {
      return severityRank[a.severity] - severityRank[b.severity];
    }
    return b.confidence - a.confidence;
  });
  const high = sorted.filter((r) => r.confidence >= 0.9);
  const low = sorted.filter((r) => r.confidence < 0.9);

  return (
    <section className="space-y-3">
      <h3 className="text-base font-medium">风险识别（{sorted.length}）</h3>
      {high.length > 0 && (
        <ul className="space-y-2">
          {high.map((r, i) => (
            <RiskItem key={`h-${i}`} risk={r} />
          ))}
        </ul>
      )}
      {low.length > 0 && (
        <details className="rounded-md border border-zinc-200 dark:border-zinc-800">
          <summary className="cursor-pointer px-3 py-2 text-xs text-zinc-500">
            {low.length} 条低置信度建议（点击展开）
          </summary>
          <ul className="space-y-2 px-3 pb-3 pt-1">
            {low.map((r, i) => (
              <RiskItem key={`l-${i}`} risk={r} />
            ))}
          </ul>
        </details>
      )}
    </section>
  );
}

function RiskItem({ risk }: { risk: Risk }) {
  return (
    <li className="rounded-md border border-zinc-200 p-3 text-sm dark:border-zinc-800">
      <div className="flex items-start gap-2">
        <span
          className={`shrink-0 rounded px-2 py-0.5 text-xs font-medium uppercase ${severityClass[risk.severity]}`}
        >
          {risk.severity}
        </span>
        <div className="flex-1 space-y-1">
          <div className="flex items-baseline gap-2 text-xs text-zinc-600 dark:text-zinc-400">
            <code className="rounded bg-zinc-100 px-1 dark:bg-zinc-800">
              {risk.file}
              {risk.line ? `:${risk.line}` : ""}
            </code>
            <span>{risk.category}</span>
            <span>conf {risk.confidence.toFixed(2)}</span>
          </div>
          <p>{risk.reason}</p>
        </div>
      </div>
    </li>
  );
}
