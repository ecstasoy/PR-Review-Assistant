"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

import { listReviews } from "@/lib/api";
import type { ReviewSummary } from "@/lib/types";

export function HistoryList() {
  const [items, setItems] = useState<ReviewSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    listReviews()
      .then((d) => {
        if (!cancelled) setItems(d);
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (error) {
    return <p className="text-sm text-red-600 dark:text-red-400">加载历史失败：{error}</p>;
  }
  if (items === null) {
    return <p className="text-sm text-zinc-500">加载中…</p>;
  }
  if (items.length === 0) {
    return (
      <p className="text-sm text-zinc-500">尚无评审记录。提交一个 PR 后会出现在这里。</p>
    );
  }
  return (
    <ul className="space-y-2">
      {items.map((item) => (
        <li key={item.id}>
          <Link
            href={`/review/${item.id}`}
            className="block rounded-md border border-zinc-200 p-4 transition-colors hover:border-zinc-400 dark:border-zinc-800 dark:hover:border-zinc-600"
          >
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <h2 className="truncate text-sm font-medium">
                  {item.title || `${item.owner}/${item.repo}#${item.pr}`}
                </h2>
                <p className="mt-1 text-xs text-zinc-500">
                  {item.owner}/{item.repo}#{item.pr} ·{" "}
                  <code className="rounded bg-zinc-100 px-1 dark:bg-zinc-800">
                    {item.head_sha.slice(0, 7)}
                  </code>
                </p>
              </div>
              <time className="shrink-0 text-xs text-zinc-400">
                {formatDate(item.created_at)}
              </time>
            </div>
          </Link>
        </li>
      ))}
    </ul>
  );
}

// formatDate 把 RFC3339 转成更友好的本地时间；解析失败时原样返回
function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString("zh-CN", { hour12: false });
}
