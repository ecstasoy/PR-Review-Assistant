"use client";

import { Lock } from "lucide-react";

import { signInURL } from "@/lib/auth";

// LoginGateBanner 替代 UrlInputCard 给未登录用户看
// 解释为何要登录 + 主按钮跳 GitHub OAuth；登录后自动跳回落地页
export function LoginGateBanner() {
  return (
    <div className="rounded-lg border border-border-strong bg-surface p-6 shadow-md">
      <div className="flex items-start gap-3">
        <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-accent-soft text-accent">
          <Lock className="h-4 w-4" />
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="mb-1.5 text-base font-semibold">登录后即可提交评审</h2>
          <p className="mb-4 text-sm text-text-2">
            评审会消耗 LLM 调用配额；登录后我们能把评审记录归到你账号下，方便你随时回看 / 删除。
            登录不需要授权写权限——只读取你的 GitHub login 和头像。
          </p>
          <a
            href={signInURL()}
            className="inline-flex h-9 items-center gap-1.5 rounded-md bg-accent px-3.5 text-sm font-medium text-accent-fg hover:opacity-90"
          >
            <GitHubMark className="h-4 w-4" />
            用 GitHub 账号登录
          </a>
          <p className="mt-3 text-xs text-faint">
            浏览历史评审记录无需登录；下面列表对所有访客可见。
          </p>
        </div>
      </div>
    </div>
  );
}

// GitHubMark 内联 SVG（避 lucide v1 没 Github icon 的问题）
function GitHubMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={className} fill="currentColor" aria-hidden>
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
    </svg>
  );
}
