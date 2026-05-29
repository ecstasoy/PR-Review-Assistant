"use client";

import { CornerDownLeft, ExternalLink, GitPullRequest, Sparkles } from "lucide-react";

import { cn } from "@/lib/utils";

// 设计原型示例 chips：3 条公开 PR 链接快速填入
const EXAMPLES = [
  {
    label: "golang/go",
    url: "https://github.com/golang/go/pull/67890",
    desc: "Go · runtime GC",
  },
  {
    label: "vercel/next.js",
    url: "https://github.com/vercel/next.js/pull/71402",
    desc: "Rust · turbopack",
  },
  {
    label: "fastapi/fastapi",
    url: "https://github.com/fastapi/fastapi/pull/12056",
    desc: "Python · DI",
  },
] as const;

// 校验 PR URL 形状（owner/repo/pull/编号），允许 /files 后缀和末尾斜杠
const PR_URL_REGEX = /github\.com\/[^/]+\/[^/]+\/pull\/\d+/;

export function isValidPrUrl(url: string): boolean {
  return PR_URL_REGEX.test(url.trim());
}

interface Props {
  value: string;
  onChange: (next: string) => void;
  onSubmit: (url: string) => void;
  disabled?: boolean;
}

// UrlInputCard URL 输入条 + 示例 chips
// 卡片样式：surface + border-strong + shadow-md（对齐原型）
export function UrlInputCard({ value, onChange, onSubmit, disabled }: Props) {
  const valid = isValidPrUrl(value);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (valid && !disabled) onSubmit(value.trim());
  }

  return (
    <div>
      <form onSubmit={submit}>
        <div className="flex items-center gap-2 rounded-lg border border-border-strong bg-surface p-2 shadow-md">
          <GitPullRequest className="ml-1.5 h-5 w-5 shrink-0 text-muted" aria-hidden />
          <input
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder="https://github.com/owner/repo/pull/123"
            spellCheck={false}
            disabled={disabled}
            className="min-w-0 flex-1 border-none bg-transparent px-0.5 py-1.5 font-mono text-sm text-text outline-none placeholder:text-faint disabled:opacity-60"
          />
          <button
            type="submit"
            disabled={!valid || disabled}
            className={cn(
              "inline-flex h-10 items-center gap-2 rounded-md px-4 text-sm font-medium transition-colors",
              "bg-accent text-accent-fg hover:opacity-90",
              "disabled:cursor-not-allowed disabled:opacity-50",
            )}
          >
            <Sparkles className="h-[15px] w-[15px]" />
            开始评审
            <CornerDownLeft className="h-[13px] w-[13px] opacity-80" />
          </button>
        </div>
      </form>

      <div className="mt-4 flex flex-wrap items-center gap-2.5">
        <span className="whitespace-nowrap text-xs text-muted">试试：</span>
        {EXAMPLES.map((ex) => (
          <button
            key={ex.url}
            type="button"
            disabled={disabled}
            onClick={() => onChange(ex.url)}
            className="inline-flex items-center gap-1.5 rounded-full border border-border bg-surface px-3 py-1 font-mono text-xs text-text-2 transition-colors hover:bg-surface-hover disabled:cursor-not-allowed disabled:opacity-50"
          >
            <ExternalLink className="h-[11px] w-[11px] text-faint" aria-hidden />
            {ex.label}
            <span className="text-faint">· {ex.desc}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
