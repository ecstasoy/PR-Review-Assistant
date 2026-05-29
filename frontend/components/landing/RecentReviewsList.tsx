"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ChevronRight, History } from "lucide-react";

import { listReviews } from "@/lib/api";
import type { ReviewSummary } from "@/lib/types";
import { CIStatus } from "@/components/ui/ci-status";
import { RiskPips } from "./RiskPips";

// ReviewSummary 在 lib/types.ts 还没含 ci / risk_counts（A3 加了后端但 type 未跟）
// 这里临时扩字段；下个清理 PR 把 lib/types.ts 同步
interface SummaryWithCounts extends ReviewSummary {
  ci?: string;
  risk_counts?: { high: number; medium: number; low: number };
}

const ZERO_COUNTS = { high: 0, medium: 0, low: 0 };

// RecentReviewsList 拉 /api/reviews?limit=4，渲染按 design 原型 4 条紧凑列表
// 失败 / 空状态用 design 的 muted 文字处理，不抛错也不显眼
export function RecentReviewsList() {
  const [items, setItems] = useState<SummaryWithCounts[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    listReviews(4)
      .then((d) => {
        if (!cancelled) setItems(d as SummaryWithCounts[]);
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <section className="mt-11">
      <div className="mb-3 flex items-center">
        <History className="mr-[7px] h-[15px] w-[15px] text-muted" aria-hidden />
        <span className="text-sm font-semibold">最近评审</span>
        <Link
          href="/history"
          className="ml-auto inline-flex items-center gap-1 text-xs text-muted hover:text-text"
        >
          查看全部 <ChevronRight className="h-3 w-3" />
        </Link>
      </div>

      <div className="overflow-hidden rounded-lg border border-border bg-surface">
        <ListBody items={items} error={error} />
      </div>
    </section>
  );
}

function ListBody({
  items,
  error,
}: {
  items: SummaryWithCounts[] | null;
  error: string | null;
}) {
  if (error) {
    return <EmptyText>加载失败：{error}</EmptyText>;
  }
  if (items === null) {
    return <EmptyText>加载中…</EmptyText>;
  }
  if (items.length === 0) {
    return <EmptyText>还没有评审记录。提交一个 PR 链接试试。</EmptyText>;
  }
  return (
    <>
      {items.map((item, i) => (
        <RecentRow key={item.id} item={item} isFirst={i === 0} />
      ))}
    </>
  );
}

function RecentRow({ item, isFirst }: { item: SummaryWithCounts; isFirst: boolean }) {
  return (
    <Link
      href={`/history/${item.id}`}
      className={`flex items-center gap-3 px-3.5 py-2.5 transition-colors hover:bg-surface-hover ${
        isFirst ? "" : "border-t border-border"
      }`}
    >
      <CIStatus status={item.ci || "pending"} />
      <code className="shrink-0 font-mono text-xs text-text-2">
        {item.owner}/{item.repo}#{item.pr}
      </code>
      <span className="flex-1 truncate text-sm text-text">
        {item.title || "(未命名)"}
      </span>
      <RiskPips counts={item.risk_counts ?? ZERO_COUNTS} />
      <span className="w-14 shrink-0 text-right text-xs text-faint">
        {formatRelative(item.created_at)}
      </span>
    </Link>
  );
}

function EmptyText({ children }: { children: React.ReactNode }) {
  return <p className="px-4 py-6 text-center text-sm text-muted">{children}</p>;
}

// formatRelative 简化版：以"刚刚 / N 分钟前 / N 小时前 / N 天前 / 日期"显示
// 不引 dayjs / date-fns，省一个依赖
function formatRelative(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const delta = (Date.now() - d.getTime()) / 1000; // seconds
  if (delta < 60) return "刚刚";
  if (delta < 3600) return `${Math.floor(delta / 60)} 分钟前`;
  if (delta < 86400) return `${Math.floor(delta / 3600)} 小时前`;
  if (delta < 7 * 86400) return `${Math.floor(delta / 86400)} 天前`;
  return d.toLocaleDateString("zh-CN", { month: "2-digit", day: "2-digit" });
}
