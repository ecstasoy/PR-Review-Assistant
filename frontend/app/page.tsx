"use client";

import { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { streamReview, type PrMeta } from "@/lib/sse";
import type { Risk, Suggestion } from "@/lib/types";
import { CapabilityCards } from "@/components/landing/CapabilityCards";
import { HeroBanner } from "@/components/landing/HeroBanner";
import { RecentReviewsList } from "@/components/landing/RecentReviewsList";
import { UrlInputCard } from "@/components/landing/UrlInputCard";
import { RisksList } from "@/components/review/RisksList";
import { SuggestionList } from "@/components/SuggestionList";
import { Spinner } from "@/components/ui/spinner";

type StageErrors = Partial<Record<"context" | "summary" | "risks" | "suggestions", string>>;

// HomePage 落地页。pre-submit 显示 hero + url 输入 + 能力卡 + 最近评审；
// submit 后切到 results 面板（最小可用，design 完整 Report 视图在 Step 3 里重做）
export default function HomePage() {
  const [url, setUrl] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [streaming, setStreaming] = useState(false);
  const [fatalError, setFatalError] = useState<string | null>(null);
  const [info, setInfo] = useState<string | null>(null);
  const [pr, setPr] = useState<PrMeta | null>(null);
  const [summary, setSummary] = useState("");
  const [risks, setRisks] = useState<Risk[]>([]);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [stageErrors, setStageErrors] = useState<StageErrors>({});
  const [risksReceived, setRisksReceived] = useState(false);
  const [suggestionsReceived, setSuggestionsReceived] = useState(false);

  async function start(target: string) {
    setSubmitted(true);
    setFatalError(null);
    setInfo(null);
    setPr(null);
    setSummary("");
    setRisks([]);
    setSuggestions([]);
    setStageErrors({});
    setRisksReceived(false);
    setSuggestionsReceived(false);
    setStreaming(true);
    try {
      await streamReview(target, {
        onPr: setPr,
        onSummaryDelta: (delta) => setSummary((s) => s + delta),
        onRisksDone: (r) => { setRisks(r); setRisksReceived(true); },
        onSuggestionsDone: (s) => { setSuggestions(s); setSuggestionsReceived(true); },
        onInfo: setInfo,
        onStageError: (stage, msg) =>
          setStageErrors((prev) => ({ ...prev, [stage]: msg })),
      });
    } catch (err) {
      setFatalError(err instanceof Error ? err.message : String(err));
    } finally {
      setStreaming(false);
    }
  }

  return (
    <section className="mx-auto -mt-8 max-w-[720px] pt-[clamp(40px,9vh,96px)] pb-16">
      <HeroBanner />
      <UrlInputCard value={url} onChange={setUrl} onSubmit={start} disabled={streaming} />
      {fatalError ? (
        <p className="mt-3 text-sm text-fail">{fatalError}</p>
      ) : null}

      {!submitted ? (
        <>
          <CapabilityCards />
          <RecentReviewsList />
        </>
      ) : (
        <ResultsPanel
          pr={pr}
          summary={summary}
          risks={risks}
          suggestions={suggestions}
          info={info}
          stageErrors={stageErrors}
          streaming={streaming}
          risksReceived={risksReceived}
          suggestionsReceived={suggestionsReceived}
        />
      )}
    </section>
  );
}

interface ResultsPanelProps {
  pr: PrMeta | null;
  summary: string;
  risks: Risk[];
  suggestions: Suggestion[];
  info: string | null;
  stageErrors: StageErrors;
  streaming: boolean;
  risksReceived: boolean;
  suggestionsReceived: boolean;
}

// ResultsPanel 占位实现：summary 流式 markdown + RiskList + SuggestionList。
// Step 3 (Report View) 会替换为完整的报告卡 + 风险分级排序 + 跨视图跳转。
function ResultsPanel({
  pr,
  summary,
  risks,
  suggestions,
  info,
  stageErrors,
  streaming,
  risksReceived,
  suggestionsReceived,
}: ResultsPanelProps) {
  if (info) {
    return (
      <div className="mt-6 rounded-md border border-border bg-surface-2 p-4 text-sm text-text-2">
        {info}
      </div>
    );
  }
  return (
    <article className="mt-6 space-y-5 rounded-lg border border-border bg-surface p-5">
      <header className="border-b border-border pb-3">
        {pr ? (
          <>
            <h2 className="text-lg font-medium">{pr.title}</h2>
            <p className="mt-1 flex flex-wrap items-center gap-x-2 font-mono text-xs text-muted">
              <span>
                {pr.owner}/{pr.repo}#{pr.pr}
              </span>
              <span>·</span>
              <code className="rounded bg-surface-2 px-1 py-0.5">
                {pr.head_sha.slice(0, 7)}
              </code>
              {streaming ? (
                <span className="ml-1 inline-flex items-center gap-1 text-faint">
                  <Spinner size="xs" /> 流式分析中…
                </span>
              ) : null}
            </p>
          </>
        ) : (
          <p className="flex items-center gap-2 text-xs text-muted">
            <Spinner size="xs" /> 等待 GitHub 响应…
          </p>
        )}
      </header>

      {stageErrors.context ? (
        <StageErrorBanner stage="上下文" message={stageErrors.context} />
      ) : null}

      <SummarySection
        summary={summary}
        streaming={streaming}
        error={stageErrors.summary}
      />
      <RisksSection
        risks={risks}
        streaming={streaming}
        error={stageErrors.risks}
        risksReceived={risksReceived}
      />
      <SuggestionsSection
        suggestions={suggestions}
        streaming={streaming}
        error={stageErrors.suggestions}
        suggestionsReceived={suggestionsReceived}
      />
    </article>
  );
}

function SummarySection({
  summary,
  streaming,
  error,
}: {
  summary: string;
  streaming: boolean;
  error: string | undefined;
}) {
  if (error) return <StageErrorBanner stage="总结" message={error} />;
  if (summary) {
    return (
      <div className="space-y-3 text-sm leading-relaxed [&_code]:rounded [&_code]:bg-surface-2 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_h1]:mt-4 [&_h1]:text-xl [&_h1]:font-semibold [&_h2]:mt-3 [&_h2]:text-lg [&_h2]:font-medium [&_li]:my-1 [&_p]:my-2 [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5">
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{summary}</ReactMarkdown>
        {streaming ? (
          <span className="inline-block h-3 w-[5px] animate-caret-blink bg-text align-middle" />
        ) : null}
      </div>
    );
  }
  if (streaming) return <p className="text-sm text-muted">生成总结中…</p>;
  return null;
}

function RisksSection({
  risks,
  streaming,
  error,
  risksReceived,
}: {
  risks: Risk[];
  streaming: boolean;
  error: string | undefined;
  risksReceived: boolean;
}) {
  if (error) return <StageErrorBanner stage="风险" message={error} />;
  if (risks.length > 0) return <RisksList risks={risks} />;
  if (risksReceived) return <p className="text-sm text-muted">未发现风险。</p>;
  if (streaming) return <p className="text-sm text-faint">扫描风险中…</p>;
  return null;
}

function SuggestionsSection({
  suggestions,
  streaming,
  error,
  suggestionsReceived,
}: {
  suggestions: Suggestion[];
  streaming: boolean;
  error: string | undefined;
  suggestionsReceived: boolean;
}) {
  if (error) return <StageErrorBanner stage="建议" message={error} />;
  if (suggestions.length > 0) return <SuggestionList suggestions={suggestions} />;
  if (suggestionsReceived) return <p className="text-sm text-muted">无建议。</p>;
  if (streaming) return <p className="text-sm text-faint">生成建议中…</p>;
  return null;
}

function StageErrorBanner({ stage, message }: { stage: string; message: string }) {
  return (
    <div className="rounded-md border border-high-bd bg-high-bg px-3 py-2 text-sm text-high">
      <span className="font-medium">{stage}失败：</span>
      {message}
    </div>
  );
}
