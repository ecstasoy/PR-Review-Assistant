import { cn } from "@/lib/utils";

interface Props {
  size?: number; // height in px
  className?: string;
  variant?: "lockup" | "icon" | "wordmark";
  animate?: boolean; // 是否闪烁光标块
}

// BrandMark LGTM 品牌标识。
// 三个 variant：lockup 图标+字标 / icon 仅图标 / wordmark 仅字标。
// 主题适应：light 黑底白字 + 鲜绿光标；dark 浅底深字 + 哑光绿光标。
// 色值取自 prototype/brand/README.md（产品原创资产）。
export function BrandMark({
  size = 26,
  className,
  variant = "lockup",
  animate = false,
}: Props) {
  const icon = (
    <svg width={size} height={size} viewBox="0 0 96 96" aria-hidden>
      {/* 圆角方块：light=深 / dark=浅，对齐 prototype/brand 的 lockup 反色规则 */}
      <rect
        width="96"
        height="96"
        rx="22"
        className="fill-[#18181b] dark:fill-[#f4f4f5]"
      />
      {/* 终端 > 提示符 */}
      <path
        d="M26 33l15 15-15 15"
        strokeWidth="8"
        fill="none"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="stroke-[#5c5c63] dark:stroke-[#8b8b93]"
      />
      {/* 光标块：固定绿（approve / merge-ready 语义） */}
      <rect
        x="50"
        y="56"
        width="22"
        height="9"
        rx="2"
        className="fill-[#3fd07f] dark:fill-[#1f8a5b]"
      />
    </svg>
  );

  // 字标尺寸：caret 高 ≈ 字高 * 0.78，宽 ≈ * 0.44，对齐 wordmark SVG 比例
  const fontPx = Math.round(size * 0.92);
  const caretH = Math.round(size * 0.72);
  const caretW = Math.round(size * 0.42);
  const wordmark = (
    <span
      className="inline-flex items-baseline gap-[0.18em] font-mono font-semibold tracking-[-0.04em] leading-none text-[#18181b] dark:text-[#f4f4f5]"
      style={{ fontSize: `${fontPx}px` }}
    >
      lgtm
      <span
        aria-hidden
        className={cn(
          "inline-block rounded-[1px] bg-[#1f8a5b] dark:bg-[#3fb950]",
          animate && "animate-caret-blink",
        )}
        style={{ height: `${caretH}px`, width: `${caretW}px` }}
      />
    </span>
  );

  if (variant === "icon") {
    return (
      <span className={cn("inline-flex items-center", className)} aria-label="LGTM">
        {icon}
      </span>
    );
  }
  if (variant === "wordmark") {
    return (
      <span className={cn("inline-flex items-center", className)} aria-label="LGTM">
        {wordmark}
      </span>
    );
  }
  return (
    <span
      className={cn("inline-flex items-center gap-2.5", className)}
      aria-label="LGTM"
    >
      {icon}
      {wordmark}
    </span>
  );
}
