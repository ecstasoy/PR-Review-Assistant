"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ExternalLink, X, Zap } from "lucide-react";

import { useNotifications, type Notification } from "@/lib/notifications";

// ToastContainer 右下角浮窗；监听新 webhook 通知，弹出 + 5s 自动消失
// 多条堆叠，按时间倒序（最新在顶）
// 不持久化已读：刷新页面就重置
export function ToastContainer() {
  const { newOnes } = useNotifications(15_000);
  const [visible, setVisible] = useState<Notification[]>([]);

  useEffect(() => {
    if (newOnes.length === 0) return;
    setVisible((prev) => {
      // 防重复（多 tick 同条都 push）
      const seen = new Set(prev.map((p) => p.id));
      const fresh = newOnes.filter((n) => !seen.has(n.id));
      return [...fresh, ...prev].slice(0, 4); // 最多同时显 4 条
    });
  }, [newOnes]);

  function dismiss(id: string) {
    setVisible((prev) => prev.filter((p) => p.id !== id));
  }

  if (visible.length === 0) return null;

  return (
    <div className="pointer-events-none fixed bottom-4 right-4 z-50 flex flex-col gap-2">
      {visible.map((n) => (
        <ToastItem key={n.id} n={n} onClose={() => dismiss(n.id)} />
      ))}
    </div>
  );
}

function ToastItem({ n, onClose }: { n: Notification; onClose: () => void }) {
  // 5s 后自动消失
  useEffect(() => {
    const t = window.setTimeout(onClose, 6500);
    return () => window.clearTimeout(t);
  }, [onClose]);

  return (
    <div className="animate-fade-up pointer-events-auto w-80 rounded-lg border border-border bg-surface shadow-lg">
      <div className="flex items-start gap-2 px-3 py-2.5">
        <span className="mt-0.5 inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-accent-soft text-accent">
          <Zap className="h-3 w-3" fill="currentColor" />
        </span>
        <div className="min-w-0 flex-1">
          <div className="text-xs font-semibold text-text">PR 已自动评审</div>
          <code className="block truncate font-mono text-[10.5px] text-text-2">
            {n.owner}/{n.repo}#{n.pr}
          </code>
          {n.title ? (
            <div className="truncate text-[11px] text-muted">{n.title}</div>
          ) : null}
          <Link
            href={`/review/${n.review_id}`}
            className="mt-1 inline-flex items-center gap-0.5 text-[11px] text-accent underline hover:opacity-80"
            onClick={onClose}
          >
            查看完整评审
            <ExternalLink className="h-2.5 w-2.5" />
          </Link>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="shrink-0 text-muted hover:text-text"
          aria-label="关闭"
        >
          <X className="h-3 w-3" />
        </button>
      </div>
    </div>
  );
}
