"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { History as HistoryIcon, Search, Trash2 } from "lucide-react";

import { listReviews } from "@/lib/api";
import type { ReviewSummary } from "@/lib/types";
import { cn } from "@/lib/utils";
import { useMe } from "@/lib/auth";
import { deleteReview } from "@/lib/reviews";
import { CIStatus, type CIStatusValue } from "@/components/ui/ci-status";
import { RiskPips } from "@/components/landing/RiskPips";

const ZERO_COUNTS = { high: 0, medium: 0, low: 0 } as const;

// HistoryPage 历史评审密集表格。
// 6 列 grid 严格对齐 design 原型 History.jsx：CI / 仓库PR / 标题 / 风险 / SHA / 时间。
// 工具行：搜索框（按 repo+title 子串）+ 语言筛选段控（从当前条目的 lang 字段动态聚合）。
// 单条点击 → /review/[id]，命中缓存秒回。
export default function HistoryPage() {
  const [items, setItems] = useState<ReviewSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [lang, setLang] = useState<string>("all");
  const [nonce, setNonce] = useState(0);
  const { me } = useMe();
  const myLogin = me?.authenticated ? me.login : undefined;

  useEffect(() => {
    let cancelled = false;
    // 拉到 maxListLimit=100，本地筛选；列表场景体积可控
    listReviews(100)
      .then((d) => {
        if (!cancelled) setItems(d);
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

  // langs 段控值 = ["all", ...当前结果集所有非空 lang 去重]
  const langs = useMemo<string[]>(() => {
    if (!items) return ["all"];
    const set = new Set<string>();
    for (const r of items) {
      if (r.lang) set.add(r.lang);
    }
    return ["all", ...Array.from(set).sort()];
  }, [items]);

  const rows = useMemo<ReviewSummary[]>(() => {
    if (!items) return [];
    const q = query.trim().toLowerCase();
    return items.filter((r) => {
      if (lang !== "all" && r.lang !== lang) return false;
      if (q === "") return true;
      const hay = `${r.owner}/${r.repo} ${r.title ?? ""}`.toLowerCase();
      return hay.includes(q);
    });
  }, [items, query, lang]);

  return (
    <section className="space-y-5">
      <header className="flex items-center gap-3">
        <HistoryIcon className="h-5 w-5 text-muted" />
        <h1 className="m-0 text-[22px] font-semibold tracking-[-0.01em]">评审历史</h1>
        <span className="font-mono text-xs text-faint">
          {items?.length ?? 0} 条 · SHA 级缓存，秒回
        </span>
      </header>

      <div className="flex flex-wrap items-center gap-2.5">
        <div className="flex h-[34px] min-w-[220px] flex-1 items-center gap-2 rounded-md border border-border-strong bg-surface px-2.5">
          <Search className="h-[15px] w-[15px] text-muted" aria-hidden />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="搜索仓库或标题…"
            className="min-w-0 flex-1 border-none bg-transparent text-sm text-text outline-none placeholder:text-muted"
          />
        </div>
        <div className="flex gap-[3px] rounded-md border border-border bg-surface p-[3px]">
          {langs.map((l) => {
            const active = lang === l;
            return (
              <button
                key={l}
                type="button"
                onClick={() => setLang(l)}
                className={cn(
                  "rounded-sm px-2.5 py-[5px] font-mono text-xs transition-colors",
                  active
                    ? "bg-surface-hover text-text"
                    : "text-muted hover:text-text",
                )}
              >
                {l}
              </button>
            );
          })}
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border border-border bg-surface">
        <HeaderRow />
        <TableBody rows={rows} items={items} error={error} myLogin={myLogin} onDelete={handleDelete} />
      </div>
    </section>
  );
}

const GRID_COLS = "grid-cols-[28px_160px_1fr_130px_90px_70px]";

function HeaderRow() {
  return (
    <div
      className={cn(
        "grid items-center gap-3 border-b border-border bg-surface-2 px-4 py-2.5",
        "text-[10.5px] font-semibold uppercase tracking-wider text-muted",
        GRID_COLS,
      )}
    >
      <span>CI</span>
      <span>仓库 / PR</span>
      <span>标题</span>
      <span>风险</span>
      <span>SHA</span>
      <span className="text-right">时间</span>
    </div>
  );
}

function TableBody({
  rows,
  items,
  error,
  myLogin,
  onDelete,
}: {
  rows: ReviewSummary[];
  items: ReviewSummary[] | null;
  error: string | null;
  myLogin?: string;
  onDelete: (id: string, label: string) => void;
}) {
  if (error) {
    return <p className="px-4 py-6 text-center text-sm text-fail">加载失败：{error}</p>;
  }
  if (items === null) {
    return <p className="px-4 py-6 text-center text-sm text-muted">加载中…</p>;
  }
  if (items.length === 0) {
    return (
      <p className="px-4 py-8 text-center text-sm text-muted">
        还没有评审记录。回到落地页提交一个 PR 链接试试。
      </p>
    );
  }
  if (rows.length === 0) {
    return <p className="px-4 py-8 text-center text-sm text-muted">无匹配结果</p>;
  }
  return (
    <>
      {rows.map((r, i) => (
        <Row
          key={r.id}
          review={r}
          isFirst={i === 0}
          myLogin={myLogin}
          onDelete={onDelete}
        />
      ))}
    </>
  );
}

function Row({
  review,
  isFirst,
  myLogin,
  onDelete,
}: {
  review: ReviewSummary;
  isFirst: boolean;
  myLogin?: string;
  onDelete: (id: string, label: string) => void;
}) {
  // 删除按钮可见性：已登录 + （我是 owner OR 匿名遗留）
  const canDelete = !!myLogin && (!review.created_by || review.created_by === myLogin);
  return (
    <div
      className={cn(
        "group relative flex items-center transition-colors hover:bg-surface-hover",
        isFirst ? "" : "border-t border-border",
      )}
    >
      <Link
        href={`/review/${review.id}`}
        className={cn(
          "grid flex-1 items-center gap-3 px-4 py-3",
          GRID_COLS,
        )}
      >
        <CIStatus status={(review.ci ?? "pending") as CIStatusValue} />
        <code className="truncate font-mono text-xs text-text-2">
          {review.owner}/{review.repo}
          <span className="text-faint">#{review.pr}</span>
        </code>
        <span className="truncate text-sm">{review.title || "(未命名)"}</span>
        <RiskPips counts={review.risk_counts ?? ZERO_COUNTS} />
        <code className="font-mono text-xs text-faint">{review.head_sha.slice(0, 7)}</code>
        <span className="text-right text-xs text-faint">{formatRelative(review.created_at)}</span>
      </Link>
      {canDelete ? (
        <button
          type="button"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onDelete(review.id, `${review.owner}/${review.repo}#${review.pr}`);
          }}
          title={review.created_by ? "删除你创建的评审" : "删除匿名遗留记录"}
          className="mr-2 inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-muted opacity-0 transition-opacity hover:bg-high-bg hover:text-high group-hover:opacity-100"
          aria-label="删除评审"
        >
          <Trash2 className="h-3.5 w-3.5" />
        </button>
      ) : null}
    </div>
  );
}

// formatRelative: 刚刚 / N 分钟前 / N 小时前 / N 天前 / MM-DD
function formatRelative(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const delta = (Date.now() - d.getTime()) / 1000;
  if (delta < 60) return "刚刚";
  if (delta < 3600) return `${Math.floor(delta / 60)} 分钟前`;
  if (delta < 86400) return `${Math.floor(delta / 3600)} 小时前`;
  if (delta < 7 * 86400) return `${Math.floor(delta / 86400)} 天前`;
  return d.toLocaleDateString("zh-CN", { month: "2-digit", day: "2-digit" });
}
