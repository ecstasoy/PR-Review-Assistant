"use client";

import { useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { streamReview, type PrMeta } from "@/lib/sse";
import type { Risk, Suggestion } from "@/lib/types";
import { RiskList } from "./RiskList";
import { SuggestionList } from "./SuggestionList";

export function ReviewForm() {
  const [url, setUrl] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pr, setPr] = useState<PrMeta | null>(null);
  const [summary, setSummary] = useState("");
  const [risks, setRisks] = useState<Risk[]>([]);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setPr(null);
    setSummary("");
    setRisks([]);
    setSuggestions([]);
    setStreaming(true);
    try {
      await streamReview(url, {
        onPr: setPr,
        onSummaryDelta: (delta) => setSummary((s) => s + delta),
        onRisksDone: setRisks,
        onSuggestionsDone: setSuggestions,
        onStageError: (stage, msg) => setError(`${stage}: ${msg}`),
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
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
        {error ? (
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        ) : null}
      </form>

      {pr || streaming ? (
        <ResultCard
          pr={pr}
          summary={summary}
          risks={risks}
          suggestions={suggestions}
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
  suggestions: Suggestion[];
  streaming: boolean;
}

function ResultCard({ pr, summary, risks, suggestions, streaming }: ResultCardProps) {
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

      {summary ? (
        <div className="space-y-3 text-sm leading-relaxed [&_code]:rounded [&_code]:bg-zinc-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_h1]:mt-4 [&_h1]:text-xl [&_h1]:font-semibold [&_h2]:mt-3 [&_h2]:text-lg [&_h2]:font-medium [&_li]:my-1 [&_p]:my-2 [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5 dark:[&_code]:bg-zinc-800">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{summary}</ReactMarkdown>
          {streaming ? (
            <span className="inline-block h-3 w-1.5 animate-pulse bg-zinc-700 align-middle dark:bg-zinc-300" />
          ) : null}
        </div>
      ) : streaming ? (
        <p className="text-sm text-zinc-500">生成总结中…</p>
      ) : null}

      {risks.length > 0 ? (
        <div className="mt-5 border-t border-zinc-200 pt-4 dark:border-zinc-800">
          <RiskList risks={risks} />
        </div>
      ) : null}

      {suggestions.length > 0 ? (
        <div className="mt-5 border-t border-zinc-200 pt-4 dark:border-zinc-800">
          <SuggestionList suggestions={suggestions} />
        </div>
      ) : null}
    </article>
  );
}
