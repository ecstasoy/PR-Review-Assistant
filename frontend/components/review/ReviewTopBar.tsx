"use client";

import Link from "next/link";
import { ChevronLeft, ExternalLink, Sparkle } from "lucide-react";

import { CIStatus, type CIStatusValue } from "@/components/ui/ci-status";
import { ThemeToggle } from "@/components/theme-toggle";

interface Props {
  title: string;
  owner: string;
  repo: string;
  pr: number;
  headSha: string;
  ci?: string; // passing / failing / pending / "" (unknown)
}

// ReviewTopBar 评审页顶栏：logo + 返回历史 + PR meta（CI 圆点 + 标题 + owner/repo#pr + head SHA）+ 原 PR 链接 + 主题切换
// 对齐 design 原型 TopBar 紧凑 + 信息密度；view switch / stage chips / 追问 dock 留下个 PR
export function ReviewTopBar({ title, owner, repo, pr, headSha, ci }: Props) {
  const githubURL = `https://github.com/${owner}/${repo}/pull/${pr}`;

  return (
    <header className="flex h-[52px] flex-shrink-0 items-center gap-3 border-b border-border bg-surface px-4">
      <Link href="/" className="flex items-center gap-2">
        <span className="inline-flex h-[26px] w-[26px] items-center justify-center rounded-md bg-accent text-accent-fg">
          <Sparkle className="h-[15px] w-[15px]" strokeWidth={2.2} fill="currentColor" />
        </span>
        <span className="hidden text-base font-semibold tracking-tight sm:inline">PR Review</span>
      </Link>

      <Link
        href="/history"
        className="inline-flex h-7 items-center gap-1 rounded-md px-2 text-xs text-muted hover:bg-surface-hover hover:text-text"
      >
        <ChevronLeft className="h-3.5 w-3.5" />
        返回历史
      </Link>

      {/* PR 信息区 —— 占据剩余主轴空间，自动溢出截断 */}
      <div className="ml-1 flex min-w-0 flex-1 items-center gap-2">
        {ci ? <CIStatus status={ci as CIStatusValue} /> : null}
        <h1 className="min-w-0 truncate text-sm font-medium" title={title}>
          {title}
        </h1>
        <span className="hidden shrink-0 font-mono text-xs text-text-2 md:inline">
          {owner}/{repo}#{pr}
        </span>
        <code className="hidden shrink-0 rounded bg-surface-2 px-1.5 py-0.5 font-mono text-[11px] text-muted md:inline">
          {headSha.slice(0, 7)}
        </code>
      </div>

      <a
        href={githubURL}
        target="_blank"
        rel="noreferrer"
        className="inline-flex h-7 items-center gap-1 rounded-md border border-border-strong bg-surface px-2.5 text-xs text-text-2 hover:bg-surface-hover hover:text-text"
      >
        查看原 PR
        <ExternalLink className="h-3 w-3" />
      </a>
      <ThemeToggle />
    </header>
  );
}
