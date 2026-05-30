"use client";

import { Suspense, use, useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

import { getReview } from "@/lib/api";
import type { File, PrMeta, Risk, ReviewDetail } from "@/lib/types";
import { ReviewTopBar, type ViewKey } from "@/components/review/ReviewTopBar";
import { Sidebar } from "@/components/review/Sidebar";
import { SummaryCard } from "@/components/review/SummaryCard";
import { RisksList } from "@/components/review/RisksList";
import { DiffView } from "@/components/review/DiffView";
import { AgentPanel } from "@/components/review/AgentPanel";
import { Spinner } from "@/components/ui/spinner";

interface PageProps {
  params: Promise<{ id: string }>;
}

const VALID_VIEWS: ViewKey[] = ["report", "diff", "session"];

// /review/[id] 评审结果页（cached-only）。
// 严格按 design 原型 ReviewResult.jsx：全宽 dashboard，顶 ReviewTopBar，
// 左 Sidebar（可折叠），中主区（max-w 1080），右 AgentPanel（可 toggle）。
// view 通过 ?view= 切换：报告 / Diff / 会话；cached 模式 stage 全 done。
// 跨视图跳转：点 Sidebar 文件 / 风险 → 切到 Diff 视图 + scrollTop 定位锚点行。
export default function ReviewDetailPage({ params }: PageProps) {
  const { id } = use(params);
  return (
    <Suspense fallback={<LoadingState />}>
      <ReviewDetailPageContent id={id} />
    </Suspense>
  );
}

function ReviewDetailPageContent({ id }: { id: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const viewParam = searchParams.get("view") as ViewKey | null;
  const view: ViewKey =
    viewParam && VALID_VIEWS.includes(viewParam) ? viewParam : "report";

  const [detail, setDetail] = useState<ReviewDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [agentOpen, setAgentOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [activeFile, setActiveFile] = useState<string | undefined>(undefined);
  const [expandRequest, setExpandRequest] = useState<{ path: string; nonce: number } | null>(null);

  const scrollRef = useRef<HTMLElement>(null);
  // 切到 Diff 视图前记下要滚的锚；视图挂载后 useEffect 消费
  const pendingScroll = useRef<string | null>(null);

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

  // scrollToAnchor 容器内手动 scrollTop 赋值（design README §6 明确不要 scrollIntoView）
  // + 1.1s 闪烁背景突出目标行
  const scrollToAnchor = useCallback((anchorId: string) => {
    const cont = scrollRef.current;
    const el = document.getElementById(anchorId);
    if (!cont || !el) return false;
    const top =
      el.getBoundingClientRect().top - cont.getBoundingClientRect().top + cont.scrollTop - 90;
    cont.scrollTop = Math.max(0, top);
    const prevBg = el.style.background;
    el.style.transition = "background .2s";
    el.style.background = "var(--accent-soft)";
    window.setTimeout(() => {
      el.style.background = prevBg;
    }, 1100);
    return true;
  }, []);

  // 视图切到 Diff 时，flush 之前积压的滚动请求
  useEffect(() => {
    if (view !== "diff" || !pendingScroll.current) return;
    const anchorId = pendingScroll.current;
    let cancelled = false;
    let attempts = 0;

    const flush = () => {
      if (cancelled) return;
      if (scrollToAnchor(anchorId)) {
        pendingScroll.current = null;
        return;
      }
      attempts += 1;
      if (attempts < 8) requestAnimationFrame(flush);
    };

    // 等 DOM mount 完成
    requestAnimationFrame(flush);
    return () => {
      cancelled = true;
    };
  }, [expandRequest?.nonce, view, scrollToAnchor]);

  function gotoView(next: ViewKey) {
    const params = new URLSearchParams(searchParams.toString());
    params.set("view", next);
    router.replace(`${pathname}?${params.toString()}`, { scroll: false });
  }

  function queueDiffJump(path: string, anchor: string) {
    setActiveFile(path);
    setExpandRequest((prev) => ({ path, nonce: (prev?.nonce ?? 0) + 1 }));
    pendingScroll.current = anchor;
    if (view !== "diff") gotoView("diff");
  }

  function pickFile(path: string) {
    queueDiffJump(path, `file-${path}`);
  }

  function pickRisk(r: Risk) {
    if (r.line == null) return;
    queueDiffJump(r.file, `L-${r.file}-${r.line}`);
  }

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
    return <LoadingState />;
  }

  const pr: PrMeta = {
    id: detail.id,
    owner: detail.owner,
    repo: detail.repo,
    pr: detail.pr,
    url: `https://github.com/${detail.owner}/${detail.repo}/pull/${detail.pr}`,
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
  const files: File[] = detail.files ?? [];
  const risks: Risk[] = detail.risks ?? [];

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
      <div className="flex min-h-0 flex-1">
        {!sidebarCollapsed && view !== "session" ? (
          <Sidebar
            pr={pr}
            files={files}
            risks={risks}
            activeFile={activeFile}
            onPickFile={pickFile}
            onPickRisk={pickRisk}
          />
        ) : null}
        <main ref={scrollRef} className="min-w-0 flex-1 overflow-y-auto">
          <div className="mx-auto flex max-w-[1080px] flex-col gap-4 px-5 py-5">
            {view === "report" ? (
              <>
                <SummaryCard summary={detail.summary} />
                {risks.length > 0 ? <RisksList risks={risks} /> : null}
              </>
            ) : view === "diff" ? (
              <DiffView
                files={files}
                risks={risks}
                suggestions={detail.suggestions}
                expandedFilePath={expandRequest?.path}
                expandedFileNonce={expandRequest?.nonce}
              />
            ) : (
              <SessionStub />
            )}
          </div>
        </main>
        {agentOpen ? <AgentPanel onClose={() => setAgentOpen(false)} /> : null}
      </div>
    </div>
  );
}

function LoadingState() {
  return (
    <p className="flex items-center gap-2 px-6 py-8 text-sm text-muted">
      <Spinner size="xs" /> 加载中…
    </p>
  );
}

function SessionStub() {
  return (
    <section className="rounded-lg border border-border bg-surface p-8 text-center">
      <p className="text-sm font-medium text-text">会话视图即将上线</p>
      <p className="mt-2 text-xs text-muted">
        agent 步骤时间线 + 实时引导 —— v2 落地。
      </p>
    </section>
  );
}
