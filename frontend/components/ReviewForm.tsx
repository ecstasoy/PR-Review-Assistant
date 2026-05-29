"use client";

import { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { streamReview, type PrMeta } from "@/lib/sse";
import type { Risk, Suggestion } from "@/lib/types";
import { RiskList } from "./RiskList";
import { SuggestionList } from "./SuggestionList";

type StageErrors = Partial<Record<"context" | "summary" | "risks" | "suggestions", string>>;

export function ReviewForm() {
  const [url, setUrl] = useState("");
  const [streaming, setStreaming] = useState(false);
  // fatalError 用于网络 / 4xx / 5xx 等致命错误，置顶展示
  const [fatalError, setFatalError] = useState<string | null>(null);
  // info 用于后端短路（如空 PR），独立蓝条展示
  const [info, setInfo] = useState<string | null>(null);
  const [pr, setPr] = useState<PrMeta | null>(null);
  const [summary, setSummary] = useState("");
  const [risks, setRisks] = useState<Risk[]>([]);
  const [risksReceived, setRisksReceived] = useState(false);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);
  const [suggestionsReceived, setSuggestionsReceived] = useState(false);
  const [stageErrors, setStageErrors] = useState<StageErrors>({});

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setFatalError(null);
    setInfo(null);
    setPr(null);
    setSummary("");
    setRisks([]);
    setRisksReceived(false);
    setSuggestions([]);
    setSuggestionsReceived(false);
    setStageErrors({});
    setStreaming(true);
    try {
      await streamReview(url, {
        onPr: setPr,
        onSummaryDelta: (delta) => setSummary((s) => s + delta),
        onRisksDone: (r) => {
          setRisks(r);
          setRisksReceived(true);
        },
        onSuggestionsDone: (s) => {
          setSuggestions(s);
          setSuggestionsReceived(true);
        },
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
    <div className="space-y-6">
      <form onSubmit={onSubmit} className="space-y-4">
        <label className="block">
          <span className="block text-sm font-medium">GitHub PR URL</span>
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://github.com/owner/repo/pull/123"
            disabled={streaming}
            className="mt-1 w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm shadow-sm outline-none focus:border-zinc-900 disabled:opacity-60 dark:border-zinc-700 dark:bg-zinc-900 dark:focus:border-zinc-100"
            required
          />
        </label>
        <button
          type="submit"
          disabled={streaming || !url}
          className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
        >
          {streaming ? "分析中…" : "开始评审"}
        </button>
        {fatalError ? (
          <p className="text-sm text-red-600 dark:text-red-400">{fatalError}</p>
        ) : null}
      </form>

      {pr || streaming ? (
        <ResultCard
          pr={pr}
          summary={summary}
          risks={risks}
          risksReceived={risksReceived}
          suggestions={suggestions}
          suggestionsReceived={suggestionsReceived}
          info={info}
          stageErrors={stageErrors}
          streaming={streaming}
        />
      ) : null}
    </div>
  );
}

interface ResultCardProps {
  pr: PrMeta | null;
  summary: string;
  risks: Risk[];
  risksReceived: boolean;
  suggestions: Suggestion[];
  suggestionsReceived: boolean;
  info: string | null;
  stageErrors: StageErrors;
  streaming: boolean;
}

function ResultCard({
  pr,
  summary,
  risks,
  risksReceived,
  suggestions,
  suggestionsReceived,
  info,
  stageErrors,
  streaming,
}: ResultCardProps) {
  // 后端短路时（空 PR），不显示各 stage 占位 / 空状态
  const shortCircuited = info !== null;

  return (
    <article className="rounded-lg border border-zinc-200 p-5 dark:border-zinc-800">
      <header className="mb-3 border-b border-zinc-200 pb-2 dark:border-zinc-800">
        {pr ? (
          <>
            <h2 className="text-lg font-medium">{pr.title}</h2>
            <p className="mt-1 text-xs text-zinc-500">
              {pr.owner}/{pr.repo}#{pr.pr} ·{" "}
              <code className="rounded bg-zinc-100 px-1 dark:bg-zinc-800">
                {pr.head_sha.slice(0, 7)}
              </code>
              {streaming ? <span className="ml-2 text-zinc-400">流式分析中…</span> : null}
            </p>
          </>
        ) : (
          <p className="text-xs text-zinc-500">等待 GitHub 响应…</p>
        )}
      </header>

      {info ? (
        <div className="rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-800 dark:border-blue-900 dark:bg-blue-950 dark:text-blue-200">
          {info}
        </div>
      ) : null}

      {!shortCircuited ? (
        <>
          {stageErrors.context ? (
            <StageErrorBanner stage="上下文" message={stageErrors.context} />
          ) : null}
          <SummarySection
            summary={summary}
            streaming={streaming}
            error={stageErrors.summary}
          />
          <Section
            title="风险"
            received={risksReceived}
            empty={risks.length === 0}
            streaming={streaming}
            error={stageErrors.risks}
            emptyLabel="未发现风险"
            streamingLabel="扫描风险中…"
          >
            <RiskList risks={risks} />
          </Section>
          <Section
            title="建议"
            received={suggestionsReceived}
            empty={suggestions.length === 0}
            streaming={streaming}
            error={stageErrors.suggestions}
            emptyLabel="未发现可改进点"
            streamingLabel="生成建议中…"
          >
            <SuggestionList suggestions={suggestions} />
          </Section>
        </>
      ) : null}
    </article>
  );
}

interface SummarySectionProps {
  summary: string;
  streaming: boolean;
  error: string | undefined;
}

function SummarySection({ summary, streaming, error }: SummarySectionProps) {
  if (error) {
    return <StageErrorBanner stage="总结" message={error} />;
  }
  if (summary) {
    return (
      <div className="space-y-3 text-sm leading-relaxed [&_code]:rounded [&_code]:bg-zinc-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_h1]:mt-4 [&_h1]:text-xl [&_h1]:font-semibold [&_h2]:mt-3 [&_h2]:text-lg [&_h2]:font-medium [&_li]:my-1 [&_p]:my-2 [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5 dark:[&_code]:bg-zinc-800">
        <ReactMarkdown remarkPlugins={[remarkGfm]}>{summary}</ReactMarkdown>
        {streaming ? (
          <span className="inline-block h-3 w-1.5 animate-pulse bg-zinc-700 align-middle dark:bg-zinc-300" />
        ) : null}
      </div>
    );
  }
  if (streaming) {
    return <p className="text-sm text-zinc-500">生成总结中…</p>;
  }
  return null;
}

interface SectionProps {
  title: string;
  received: boolean;
  empty: boolean;
  streaming: boolean;
  error: string | undefined;
  emptyLabel: string;
  streamingLabel: string;
  children: React.ReactNode;
}

// Section 统一处理 risks / suggestions 的四种态：error / streaming / received-empty / received-with-data
function Section({
  title,
  received,
  empty,
  streaming,
  error,
  emptyLabel,
  streamingLabel,
  children,
}: SectionProps) {
  let body: React.ReactNode = null;
  if (error) {
    body = <StageErrorBanner stage={title} message={error} />;
  } else if (received && !empty) {
    body = children;
  } else if (received && empty) {
    body = (
      <p className="text-sm text-emerald-700 dark:text-emerald-400">✓ {emptyLabel}</p>
    );
  } else if (streaming) {
    body = <p className="text-sm text-zinc-400">{streamingLabel}</p>;
  }
  if (!body) return null;
  return (
    <div className="mt-5 border-t border-zinc-200 pt-4 dark:border-zinc-800">
      {body}
    </div>
  );
}

function StageErrorBanner({ stage, message }: { stage: string; message: string }) {
  return (
    <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
      <span className="font-medium">{stage}失败：</span>
      {message}
    </div>
  );
}
