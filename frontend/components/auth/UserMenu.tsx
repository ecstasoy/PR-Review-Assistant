"use client";

import { LogOut } from "lucide-react";

import { signInURL, signOut, useMe } from "@/lib/auth";
import { cn } from "@/lib/utils";

// GitHubIcon 内联官方 mark；lucide v1 没 Github icon，避免引新依赖
function GitHubIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 16 16" className={className} fill="currentColor" aria-hidden>
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
    </svg>
  );
}

// UserMenu 顶栏右上角组件：
//   - loading：占位骨架（避免 flash）
//   - 未登录：GitHub 登录按钮（链到 /api/auth/github/login，next=当前页）
//   - 已登录：头像 + 登录名 + 登出按钮
//
// 体积刻意压紧，跟现有 ThemeToggle / "追问" 按钮一行内对齐
export function UserMenu({ className }: { className?: string }) {
  const { me, loading } = useMe();

  if (loading) {
    return (
      <div
        className={cn(
          "inline-flex h-7 w-20 animate-pulse rounded-md bg-surface-2",
          className,
        )}
        aria-hidden
      />
    );
  }

  if (!me?.authenticated) {
    return (
      <a
        href={signInURL()}
        className={cn(
          "inline-flex h-7 shrink-0 items-center gap-1.5 rounded-md border border-border-strong bg-surface px-2.5 text-xs text-text-2 hover:bg-surface-hover hover:text-text",
          className,
        )}
      >
        <GitHubIcon className="h-3.5 w-3.5" />
        GitHub 登录
      </a>
    );
  }

  return (
    <div className={cn("inline-flex items-center gap-1.5", className)}>
      <div className="inline-flex h-7 items-center gap-1.5 rounded-md border border-border bg-surface-2 px-2 text-xs">
        {me.avatar_url ? (
          // 不用 next/image 避免域名白名单配置；头像最多 26px 不在意优化
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={me.avatar_url}
            alt={me.login ?? "user"}
            className="h-5 w-5 rounded-full"
          />
        ) : null}
        <span className="font-mono text-[11px] text-text-2">{me.login}</span>
      </div>
      <button
        type="button"
        onClick={() => void signOut()}
        title="登出"
        aria-label="登出"
        className="inline-flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-surface-hover hover:text-text"
      >
        <LogOut className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
