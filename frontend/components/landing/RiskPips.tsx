// RiskPips 按 severity 渲染小圆点 + 数字（high 红 / medium 黄 / low 灰）
// 0 计数的级别不渲染，避免视觉噪声

const META: Record<"high" | "medium" | "low", { dot: string; text: string }> = {
  high: { dot: "bg-high", text: "text-high" },
  medium: { dot: "bg-med", text: "text-med" },
  low: { dot: "bg-low", text: "text-low" },
};

export interface RiskCountsValue {
  high: number;
  medium: number;
  low: number;
}

export function RiskPips({ counts }: { counts: RiskCountsValue }) {
  const items: Array<["high" | "medium" | "low", number]> = [
    ["high", counts.high],
    ["medium", counts.medium],
    ["low", counts.low],
  ];
  return (
    <span className="inline-flex shrink-0 items-center gap-1.5">
      {items.map(([sev, n]) =>
        n > 0 ? (
          <span
            key={sev}
            className={`inline-flex items-center gap-[3px] font-mono text-[10px] ${META[sev].text}`}
          >
            <span className={`inline-block h-[7px] w-[7px] rounded-full ${META[sev].dot}`} />
            {n}
          </span>
        ) : null,
      )}
    </span>
  );
}
