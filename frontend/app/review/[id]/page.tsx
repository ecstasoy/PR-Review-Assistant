"use client";

import { Suspense, use, useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

import { getReview } from "@/lib/api";
import { streamReview } from "@/lib/sse";
import type { File, PrMeta, Risk, ReviewDetail, Suggestion } from "@/lib/types";
import { ReviewTopBar, type ViewKey } from "@/components/review/ReviewTopBar";
import { Sidebar } from "@/components/review/Sidebar";
import { SessionList } from "@/components/review/SessionList";
import { SummaryCard } from "@/components/review/SummaryCard";
import { RisksList } from "@/components/review/RisksList";
import { DiffView } from "@/components/review/DiffView";
import { AgentPanel } from "@/components/review/AgentPanel";
import { AgentSessionView } from "@/components/review/AgentSessionView";
import { Spinner } from "@/components/ui/spinner";
import type { StageState } from "@/components/review/StageChip";

interface PageProps {
  params: Promise<{ id: string }>;
}

type StageErrors = Partial<Record<"context" | "summary" | "risks" | "suggestions", string>>;

const VALID_VIEWS: ViewKey[] = ["report", "diff", "session"];

// /review/[id] 评审结果页。两种模式：
// - id === "streaming" + ?url= → 实时 SSE 流式（landing submit 后跳进来）
// - id 是 ULID → cached 模式，调 getReview 拉详情
// 两种模式共享同一套 UI（顶栏 / Sidebar / 主区 / Agent dock）；StageChips 在流式时按事件推进，
// 在 cached 模式全 done。
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

  const isStreaming = id === "streaming";
  const sourceURL = searchParams.get("url");

  // 统一状态形状：cached 模式一次填齐，streaming 模式逐步填
  const [pr, setPr] = useState<PrMeta | null>(null);
  const [summary, setSummary] = useState("");
  const [risks, setRisks] = useState<Risk[]>([]);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [files, setFiles] = useState<File[]>([]);
  const [summaryDone, setSummaryDone] = useState(false);
  const [risksDone, setRisksDone] = useState(false);
  const [suggestionsDone, setSuggestionsDone] = useState(false);
  const [streaming, setStreaming] = useState(isStreaming);
  const [info, setInfo] = useState<string | null>(null);
  const [stageErrors, setStageErrors] = useState<StageErrors>({});
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);

  const [agentOpen, setAgentOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [activeFile, setActiveFile] = useState<string | undefined>(undefined);
  const [expandRequest, setExpandRequest] = useState<{ path: string; nonce: number } | null>(null);

  const scrollRef = useRef<HTMLElement>(null);
  const pendingScroll = useRef<string | null>(null);

  // 拉数据：cached 模式 getReview；streaming 模式 streamReview
  useEffect(() => {
    let cancelled = false;
    let controller: AbortController | null = null;
    if (isStreaming) {
      if (!sourceURL) {
        setError("缺少 url 参数");
        setLoaded(true);
        return;
      }
      setSummaryDone(false);
      controller = new AbortController();
      streamReview(sourceURL, {
        onPr: (p) => !cancelled && (setPr(p), setLoaded(true)),
        onFiles: (f) => !cancelled && setFiles(f),
        onSummaryDelta: (d) => !cancelled && setSummary((s) => s + d),
        onRisksDone: (r) => {
          if (cancelled) return;
          setRisks(r);
          setRisksDone(true);
        },
        onSuggestionsDone: (s) => {
          if (cancelled) return;
          setSuggestions(s);
          setSuggestionsDone(true);
        },
        onInfo: (m) => !cancelled && setInfo(m),
        onStageError: (stage, msg) => {
          if (cancelled) return;
          if (stage === "summary") setSummaryDone(true);
          setStageErrors((prev) => ({ ...prev, [stage]: msg }));
        },
        onStageDone: (stage) => {
          if (cancelled || stage !== "summary") return;
          setSummaryDone(true);
        },
        onDone: () => !cancelled && (setSummaryDone(true), setStreaming(false)),
      })
        .catch((e) => {
          if (e instanceof DOMException && e.name === "AbortError") return;
          if (!cancelled) setError(e instanceof Error ? e.message : String(e));
        })
        .finally(() => {
          if (!cancelled) {
            setStreaming(false);
            setLoaded(true);
          }
        });
    } else {
      // cached
      getReview(id)
        .then((d) => {
          if (cancelled) return;
          hydrateFromDetail(d, {
            setPr,
            setSummary,
            setRisks,
            setSuggestions,
            setFiles,
            setSummaryDone,
            setRisksDone,
            setSuggestionsDone,
            setStreaming,
          });
          setLoaded(true);
        })
        .catch((e) => {
          if (!cancelled) {
            setError(e instanceof Error ? e.message : String(e));
            setLoaded(true);
          }
        });
    }
    return () => {
      cancelled = true;
      controller?.abort();
    };
  }, [id, isStreaming, sourceURL]);

  // stage 状态机：基于 summary 是否有内容 / risksDone / suggestionsDone / streaming
  const stageStates = useMemo<{
    summary: StageState;
    risks: StageState;
    suggestions: StageState;
  }>(() => {
    if (!streaming) {
      // 流式结束 / cached → 全部按 done 显示
      return { summary: "done", risks: "done", suggestions: "done" };
    }
    const hasSummary = summary.length > 0;
    return {
      summary: summaryDone ? "done" : hasSummary ? "active" : "pending",
      risks: risksDone ? "done" : hasSummary ? "active" : "pending",
      suggestions: suggestionsDone ? "done" : risksDone ? "active" : "pending",
    };
  }, [streaming, summary, summaryDone, risksDone, suggestionsDone]);

  // scrollTop 直接赋值定位锚点 + 1.1s 闪烁高亮（design README §6 要求）
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

  // streaming 模式：pr 事件未到时显加载占位，但顶栏不渲染（无 pr meta）
  if (!loaded || !pr) {
    return <LoadingState />;
  }

  return (
    <div className="flex h-screen flex-col bg-bg">
      <ReviewTopBar
        pr={pr}
        view={view}
        stageStates={stageStates}
        onSidebarToggle={() => setSidebarCollapsed((c) => !c)}
        onToggleAgent={() => setAgentOpen((o) => !o)}
        agentOpen={agentOpen}
      />
      <div className="flex min-h-0 flex-1">
        {sidebarCollapsed ? null : view === "session" ? (
          <SessionList activeId={isStreaming ? undefined : id} />
        ) : (
          <Sidebar
            pr={pr}
            files={files}
            risks={risks}
            activeFile={activeFile}
            onPickFile={pickFile}
            onPickRisk={pickRisk}
          />
        )}
        <main ref={scrollRef} className="min-w-0 flex-1 overflow-y-auto">
          <div className="mx-auto flex max-w-[1080px] flex-col gap-4 px-5 py-5">
            {info ? (
              <div className="rounded-md border border-border bg-surface-2 px-4 py-3 text-sm text-text-2">
                {info}
              </div>
            ) : null}
            {stageErrors.context ? (
              <StageErrorBanner stage="上下文" message={stageErrors.context} />
            ) : null}
            {stageErrors.suggestions ? (
              <StageErrorBanner stage="建议" message={stageErrors.suggestions} />
            ) : null}
            {view === "report" ? (
              <ReportContent
                summary={summary}
                summaryDone={summaryDone}
                risks={risks}
                risksDone={risksDone}
                streaming={streaming}
                stageErrors={stageErrors}
              />
            ) : view === "diff" ? (
              <DiffView
                files={files}
                risks={risks}
                suggestions={suggestions}
                expandedFilePath={expandRequest?.path}
                expandedFileNonce={expandRequest?.nonce}
              />
            ) : (
              <AgentSessionView
                pr={pr}
                files={files}
                risks={risks}
                suggestions={suggestions}
                summary={summary}
                hasFiles={files.length > 0}
                risksDone={risksDone}
                suggestionsDone={suggestionsDone}
                streaming={streaming}
              />
            )}
          </div>
        </main>
        {agentOpen ? <AgentPanel onClose={() => setAgentOpen(false)} /> : null}
      </div>
    </div>
  );
}

interface HydrateSetters {
  setPr: (pr: PrMeta) => void;
  setSummary: (s: string) => void;
  setRisks: (r: Risk[]) => void;
  setSuggestions: (s: Suggestion[]) => void;
  setFiles: (f: File[]) => void;
  setSummaryDone: (b: boolean) => void;
  setRisksDone: (b: boolean) => void;
  setSuggestionsDone: (b: boolean) => void;
  setStreaming: (b: boolean) => void;
}

// hydrateFromDetail 把 cached detail 一次填齐到所有 state
function hydrateFromDetail(d: ReviewDetail, h: HydrateSetters) {
  h.setPr({
    id: d.id,
    owner: d.owner,
    repo: d.repo,
    pr: d.pr,
    url: `https://github.com/${d.owner}/${d.repo}/pull/${d.pr}`,
    head_sha: d.head_sha,
    title: d.title ?? "",
    author: d.author,
    author_role: d.author_role,
    state: d.state,
    labels: d.labels,
    base_ref: d.base_ref,
    head_ref: d.head_ref,
    pr_created_at: d.pr_created_at,
    stats: d.stats,
    ci: d.ci,
    checks: d.checks,
  });
  h.setSummary(d.summary ?? "");
  h.setRisks(d.risks ?? []);
  h.setSuggestions(d.suggestions ?? []);
  h.setFiles(d.files ?? []);
  h.setSummaryDone(true);
  h.setRisksDone(true);
  h.setSuggestionsDone(true);
  h.setStreaming(false);
}

function ReportContent({
  summary,
  summaryDone,
  risks,
  risksDone,
  streaming,
  stageErrors,
}: {
  summary: string;
  summaryDone: boolean;
  risks: Risk[];
  risksDone: boolean;
  streaming: boolean;
  stageErrors: StageErrors;
}) {
  return (
    <>
      {stageErrors.summary ? (
        <StageErrorBanner stage="总结" message={stageErrors.summary} />
      ) : (
        <SummaryCard summary={summary} streaming={streaming && !summaryDone} />
      )}
      {stageErrors.risks ? (
        <StageErrorBanner stage="风险" message={stageErrors.risks} />
      ) : risks.length > 0 ? (
        <RisksList risks={risks} />
      ) : risksDone ? (
        <p className="text-sm text-muted">未发现风险。</p>
      ) : streaming ? (
        <p className="text-sm text-faint">扫描风险中…</p>
      ) : null}
    </>
  );
}

function StageErrorBanner({ stage, message }: { stage: string; message: string }) {
  return (
    <div className="rounded-md border border-high-bd bg-high-bg px-3 py-2 text-sm text-high">
      <span className="font-medium">{stage}失败：</span>
      {message}
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

