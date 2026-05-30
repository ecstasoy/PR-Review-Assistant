"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";

import { getReview } from "@/lib/api";
import type { PrMeta, ReviewDetail } from "@/lib/types";
import { ReviewTopBar, type ViewKey } from "@/components/review/ReviewTopBar";
import { SummaryCard } from "@/components/review/SummaryCard";
import { RisksList } from "@/components/review/RisksList";
import { SuggestionList } from "@/components/SuggestionList";
import { Spinner } from "@/components/ui/spinner";

interface PageProps {
  params: Promise<{ id: string }>;
}

const VALID_VIEWS: ViewKey[] = ["report", "diff", "session"];

// /review/[id] 评审结果页。cached 模式所有 stage 默认 done。
// Sidebar / DiffView / AgentPanel 等组件由后续 commit 接入。
export default function ReviewDetailPage({ params }: PageProps) {
  const { id } = use(params);
  const searchParams = useSearchParams();
  const viewParam = searchParams.get("view") as ViewKey | null;
  const view: ViewKey =
    viewParam && VALID_VIEWS.includes(viewParam) ? viewParam : "report";

  const [detail, setDetail] = useState<ReviewDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [agentOpen, setAgentOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    getReview(id)
      .then((d) => {
        if (!cancelled) setDetail(d);
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (error) {
    return (
      <section className="space-y-4 px-6 py-8">
        <Link href="/history" className="text-xs text-muted hover:text-text">
          ← 返回历史
        </Link>
        <p className="text-sm text-fail">加载失败：{error}</p>
      </section>
    );
  }
  if (!detail) {
    return (
      <p className="flex items-center gap-2 px-6 py-8 text-sm text-muted">
        <Spinner size="xs" /> 加载中…
      </p>
    );
  }

  // PrMeta 拼接：detail 已含完整 meta，但 SSE pr 事件含 url 字段而 detail 没有
  const githubURL = `https://github.com/${detail.owner}/${detail.repo}/pull/${detail.pr}`;
  const pr: PrMeta = {
    id: detail.id,
    owner: detail.owner,
    repo: detail.repo,
    pr: detail.pr,
    url: githubURL,
    head_sha: detail.head_sha,
    title: detail.title ?? "",
    author: detail.author,
    author_role: detail.author_role,
    state: detail.state,
    labels: detail.labels,
    base_ref: detail.base_ref,
    head_ref: detail.head_ref,
    pr_created_at: detail.pr_created_at,
    stats: detail.stats,
    ci: detail.ci,
    checks: detail.checks,
  };

  return (
    <div className="flex h-screen flex-col bg-bg">
      <ReviewTopBar
        pr={pr}
        view={view}
        stageStates={{ summary: "done", risks: "done", suggestions: "done" }}
        onSidebarToggle={() => setSidebarCollapsed((c) => !c)}
        onToggleAgent={() => setAgentOpen((o) => !o)}
        agentOpen={agentOpen}
      />
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto flex max-w-[1080px] flex-col gap-4 px-5 py-5">
          {/* sidebar / diff / agent 视图由后续 commit 接入；当前仅 Report 视图可用 */}
          {view !== "session" ? (
            <>
              <SummaryCard summary={detail.summary} />
              {detail.risks && detail.risks.length > 0 ? (
                <RisksList risks={detail.risks} />
              ) : null}
              {view === "report" && detail.suggestions && detail.suggestions.length > 0 ? (
                <SuggestionList suggestions={detail.suggestions} />
              ) : null}
            </>
          ) : (
            <SessionStub />
          )}
          {sidebarCollapsed ? null : null /* 临时无操作；Sidebar commit 接入 */}
        </div>
      </main>
    </div>
  );
}

function SessionStub() {
  return (
    <section className="rounded-lg border border-border bg-surface p-8 text-center">
      <p className="text-sm font-medium text-text">会话视图即将上线</p>
      <p className="mt-2 text-xs text-muted">
        agent 步骤时间线 + 实时引导 —— 后续 PR 落地。
      </p>
    </section>
  );
}
