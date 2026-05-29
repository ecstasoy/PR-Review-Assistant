"use client";

import { ExternalLink } from "lucide-react";

import { CIStatus, type CIStatusValue } from "@/components/ui/ci-status";

interface Props {
  title: string;
  owner: string;
  repo: string;
  pr: number;
  headSha: string;
  ci?: string; // passing / failing / pending / "" (unknown)
}

// ReviewTopBar 评审页 PR 头部 strip（不替代 global NavBar，作为页面内部紧凑头条）。
// CI 圆点 + 标题 + owner/repo#pr + head SHA + 原 PR 链接。
// 完整 design TopBar（view switch / stage chips / 追问 dock）留下个 PR。
export function ReviewTopBar({ title, owner, repo, pr, headSha, ci }: Props) {
  const githubURL = `https://github.com/${owner}/${repo}/pull/${pr}`;

  return (
    <header className="flex items-center gap-3 rounded-lg border border-border bg-surface px-4 py-3">
      {ci ? <CIStatus status={ci as CIStatusValue} /> : null}
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <h1 className="truncate text-base font-medium leading-tight" title={title}>
          {title}
        </h1>
        <p className="flex items-center gap-2 text-xs text-muted">
          <span className="font-mono text-text-2">
            {owner}/{repo}#{pr}
          </span>
          <span>·</span>
          <code className="rounded bg-surface-2 px-1.5 py-0.5 font-mono text-[11px]">
            {headSha.slice(0, 7)}
          </code>
        </p>
      </div>
      <a
        href={githubURL}
        target="_blank"
        rel="noreferrer"
        className="inline-flex h-8 shrink-0 items-center gap-1 rounded-md border border-border-strong bg-surface px-3 text-xs text-text-2 hover:bg-surface-hover hover:text-text"
      >
        查看原 PR
        <ExternalLink className="h-3 w-3" />
      </a>
    </header>
  );
}
