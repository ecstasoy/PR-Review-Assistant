"use client";

import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import {
  AlignLeft,
  ExternalLink,
  FileCode2,
  PanelLeftClose,
  Sparkle,
} from "lucide-react";

import type { PrMeta } from "@/lib/types";
import { cn } from "@/lib/utils";
import { CIStatus, type CIStatusValue } from "@/components/ui/ci-status";
import { ThemeToggle } from "@/components/theme-toggle";
import { StageChip, type StageState } from "./StageChip";

export type ViewKey = "report" | "diff" | "session";

interface Props {
  pr: PrMeta;
  view: ViewKey;
  stageStates: { summary: StageState; risks: StageState; suggestions: StageState };
  onSidebarToggle: () => void;
  onToggleAgent: () => void;
  agentOpen: boolean;
}

// ReviewTopBar 完整顶栏（52px 全宽）：
// logo + divider + sidebar 折叠按钮 + PR 标题块（CI + 标题 / owner/repo#pr · sha）
// + ViewSwitch 段控（报告/Diff/会话，含 icon）+ StageChips（仅非 session 视图）
// + "查看原 PR" 外链 + "追问" toggle + theme toggle。
// 严格对齐 design 原型 ReviewResult.jsx 的 header 结构。
export function ReviewTopBar({
  pr,
  view,
  stageStates,
  onSidebarToggle,
  onToggleAgent,
  agentOpen,
}: Props) {
  const githubURL = `https://github.com/${pr.owner}/${pr.repo}/pull/${pr.pr}`;
  return (
    <header className="flex h-[52px] flex-shrink-0 items-center gap-3 border-b border-border bg-surface px-3.5">
      <Link href="/" className="flex items-center">
        <span className="inline-flex h-[26px] w-[26px] items-center justify-center rounded-md bg-accent text-accent-fg">
          <Sparkle className="h-[15px] w-[15px]" strokeWidth={2.2} fill="currentColor" />
        </span>
      </Link>
      <span className="h-[22px] w-px shrink-0 bg-border" aria-hidden />
      <button
        type="button"
        onClick={onSidebarToggle}
        title="切换侧栏"
        aria-label="切换侧栏"
        className="inline-flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-surface-hover hover:text-text"
      >
        <PanelLeftClose className="h-4 w-4" />
      </button>

      {/* PR meta 块（标题 + 副标） */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          {pr.ci ? <CIStatus status={pr.ci as CIStatusValue} /> : null}
          <span className="truncate text-sm font-semibold leading-tight" title={pr.title}>
            {pr.title}
          </span>
        </div>
        <div className="flex items-center gap-2 font-mono text-[11px] text-muted">
          <span>
            {pr.owner}/{pr.repo}
            <span className="text-faint">#{pr.pr}</span>
          </span>
          <span className="text-faint">·</span>
          <code className="rounded bg-surface-2 px-[5px] text-[11px]">
            {pr.head_sha?.slice(0, 11) ?? ""}
          </code>
        </div>
      </div>

      <ViewSwitch view={view} />

      {view !== "session" ? (
        <div className="flex items-center gap-3.5 pr-1.5 pl-3">
          <StageChip label="总结" state={stageStates.summary} />
          <StageChip label="风险" state={stageStates.risks} />
          <StageChip label="建议" state={stageStates.suggestions} />
        </div>
      ) : null}

      <a
        href={githubURL}
        target="_blank"
        rel="noreferrer"
        className="inline-flex h-7 shrink-0 items-center gap-1 rounded-md border border-border-strong bg-surface px-2.5 text-xs text-text-2 hover:bg-surface-hover hover:text-text"
      >
        <ExternalLink className="h-3 w-3" />
        查看原 PR
      </a>
      <button
        type="button"
        onClick={onToggleAgent}
        className={cn(
          "inline-flex h-7 shrink-0 items-center gap-1 rounded-md px-2.5 text-xs font-medium transition-colors",
          agentOpen
            ? "bg-accent text-accent-fg hover:opacity-90"
            : "border border-border-strong bg-surface text-text-2 hover:bg-surface-hover hover:text-text",
        )}
      >
        <Sparkle className="h-3 w-3" fill={agentOpen ? "currentColor" : "none"} />
        追问
      </button>
      <ThemeToggle />
    </header>
  );
}

// ViewSwitch 内联在 TopBar 里——3 个段：报告 / Diff / 会话；
// 用 Next.js Link href=?view=... 持久化到 URL，scroll={false} 防滚动跳。
function ViewSwitch({ view }: { view: ViewKey }) {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const items: Array<{ key: ViewKey; label: string; icon: typeof AlignLeft }> = [
    { key: "report", label: "报告", icon: AlignLeft },
    { key: "diff", label: "Diff", icon: FileCode2 },
    { key: "session", label: "会话", icon: Sparkle },
  ];
  return (
    <div className="flex gap-[3px] rounded-md border border-border bg-surface-2 p-[3px]">
      {items.map(({ key, label, icon: Icon }) => {
        const active = key === view;
        const params = new URLSearchParams(searchParams.toString());
        params.set("view", key);
        return (
          <Link
            key={key}
            href={`${pathname}?${params.toString()}`}
            scroll={false}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-sm px-2.5 py-1 text-xs font-medium transition-colors",
              active
                ? "bg-surface text-text shadow-sm"
                : "text-muted hover:text-text",
            )}
          >
            <Icon
              className="h-3 w-3"
              fill={key === "session" && active ? "currentColor" : "none"}
            />
            {label}
          </Link>
        );
      })}
    </div>
  );
}
