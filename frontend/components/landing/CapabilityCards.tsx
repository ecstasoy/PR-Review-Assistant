import { AlertTriangle, AlignLeft, Sparkles } from "lucide-react";

// 三张能力卡：变更总结 / 风险识别 / 行内建议
// 30px 圆角图标块 + 标题 + 说明，grid-cols-3
const ITEMS = [
  {
    icon: AlignLeft,
    title: "变更总结",
    desc: "一段话讲清这个 PR 在做什么 + 评审重点",
  },
  {
    icon: AlertTriangle,
    title: "风险识别",
    desc: "按 severity / confidence 分级，定位到行",
  },
  {
    icon: Sparkles,
    title: "行内建议",
    desc: "锚定到 diff 行，可一键复制到 GitHub",
  },
] as const;

export function CapabilityCards() {
  return (
    <div className="mt-11 grid grid-cols-1 gap-3 sm:grid-cols-3">
      {ITEMS.map(({ icon: Icon, title, desc }) => (
        <div
          key={title}
          className="rounded-lg border border-border bg-surface p-4"
        >
          <span className="mb-2.5 inline-flex h-[30px] w-[30px] items-center justify-center rounded-lg bg-surface-2 text-accent">
            <Icon className="h-4 w-4" />
          </span>
          <div className="mb-1 text-base font-semibold">{title}</div>
          <p className="text-xs leading-[1.6] text-muted">{desc}</p>
        </div>
      ))}
    </div>
  );
}
