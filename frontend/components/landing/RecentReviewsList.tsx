"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ChevronRight, History, Trash2 } from "lucide-react";

import { listReviews } from "@/lib/api";
import type { ReviewSummary } from "@/lib/types";
import { useMe } from "@/lib/auth";
import { deleteReview } from "@/lib/reviews";
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
  const [nonce, setNonce] = useState(0); // 删除后 ++ 触发重拉
  const { me } = useMe();

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
  }, [nonce]);

  async function handleDelete(id: string, label: string) {
    if (!window.confirm(`确定删除评审「${label}」？操作不可撤销。`)) return;
    try {
      await deleteReview(id);
      setNonce((n) => n + 1);
    } catch (e) {
      window.alert("删除失败：" + (e instanceof Error ? e.message : String(e)));
    }
  }

  const myLogin = me?.authenticated ? me.login : undefined;

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
        <ListBody items={items} error={error} myLogin={myLogin} onDelete={handleDelete} />
      </div>
    </section>
  );
}

function ListBody({
  items,
  error,
  myLogin,
  onDelete,
}: {
  items: SummaryWithCounts[] | null;
  error: string | null;
  myLogin?: string;
  onDelete: (id: string, label: string) => void;
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
        <RecentRow
          key={item.id}
          item={item}
          isFirst={i === 0}
          myLogin={myLogin}
          onDelete={onDelete}
        />
      ))}
    </>
  );
}

function RecentRow({
  item,
  isFirst,
  myLogin,
  onDelete,
}: {
  item: SummaryWithCounts;
  isFirst: boolean;
  myLogin?: string;
  onDelete: (id: string, label: string) => void;
}) {
  // 删除按钮可见性：已登录 + (我是 owner OR 匿名遗留)
  // 匿名遗留（created_by 空）兼容 v1 旧记录，任何登录用户都能清
  const canDelete = !!myLogin && (!item.created_by || item.created_by === myLogin);
  return (
    <div
      className={`group relative flex items-center transition-colors hover:bg-surface-hover ${
        isFirst ? "" : "border-t border-border"
      }`}
    >
      <Link
        href={`/review/${item.id}`}
        className="flex flex-1 items-center gap-3 px-3.5 py-2.5"
      >
        <CIStatus status={item.ci || "pending"} />
        <code className="shrink-0 font-mono text-xs text-text-2">
          {item.owner}/{item.repo}#{item.pr}
        </code>
        <span className="flex-1 truncate text-sm text-text">
          {item.title || "(未命名)"}
        </span>
        {item.source === "webhook" ? (
          <span
            className="inline-flex h-[18px] shrink-0 items-center gap-0.5 rounded-full bg-accent-soft px-1.5 text-[10px] font-medium text-accent"
            title="GitHub 推 PR webhook 自动触发"
          >
            ⚡ 自动
          </span>
        ) : null}
        <RiskPips counts={item.risk_counts ?? ZERO_COUNTS} />
        <span className="w-14 shrink-0 text-right text-xs text-faint">
          {formatRelative(item.created_at)}
        </span>
      </Link>
      {canDelete ? (
        <button
          type="button"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onDelete(item.id, `${item.owner}/${item.repo}#${item.pr}`);
          }}
          title={item.created_by ? "删除你创建的评审" : "删除匿名遗留记录"}
          className="mr-2 inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted opacity-0 transition-opacity hover:bg-high-bg hover:text-high group-hover:opacity-100"
          aria-label="删除评审"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      ) : null}
    </div>
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
