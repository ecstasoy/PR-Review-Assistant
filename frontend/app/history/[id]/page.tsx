"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { getReview } from "@/lib/api";
import type { ReviewDetail } from "@/lib/types";
import { RiskList } from "@/components/RiskList";
import { SuggestionList } from "@/components/SuggestionList";

interface PageProps {
  // Next 15+ 把 dynamic route params 改成 Promise；用 React.use() 解包
  params: Promise<{ id: string }>;
}

export default function HistoryDetailPage({ params }: PageProps) {
  const { id } = use(params);
  const [detail, setDetail] = useState<ReviewDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

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
      <section className="space-y-4">
        <Link href="/history" className="text-xs text-zinc-500 hover:underline">
          ← 返回历史
        </Link>
        <p className="text-sm text-red-600 dark:text-red-400">加载失败：{error}</p>
      </section>
    );
  }
  if (!detail) {
    return <p className="text-sm text-zinc-500">加载中…</p>;
  }

  const githubURL = `https://github.com/${detail.owner}/${detail.repo}/pull/${detail.pr}`;

  return (
    <article className="space-y-5">
      <header>
        <Link href="/history" className="text-xs text-zinc-500 hover:underline">
          ← 返回历史
        </Link>
        <h1 className="mt-2 text-xl font-semibold">
          {detail.title || `${detail.owner}/${detail.repo}#${detail.pr}`}
        </h1>
        <p className="mt-1 text-xs text-zinc-500">
          <a
            href={githubURL}
            target="_blank"
            rel="noreferrer"
            className="hover:underline"
          >
            {detail.owner}/{detail.repo}#{detail.pr}
          </a>
          {" · "}
          <code className="rounded bg-zinc-100 px-1 dark:bg-zinc-800">
            {detail.head_sha.slice(0, 7)}
          </code>
          {" · "}
          {formatDate(detail.created_at)}
        </p>
      </header>

      {detail.summary ? (
        <section>
          <h2 className="mb-2 text-sm font-medium">变更总结</h2>
          <div className="space-y-3 text-sm leading-relaxed [&_code]:rounded [&_code]:bg-zinc-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_h1]:mt-4 [&_h1]:text-xl [&_h1]:font-semibold [&_h2]:mt-3 [&_h2]:text-lg [&_h2]:font-medium [&_li]:my-1 [&_p]:my-2 [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-5 dark:[&_code]:bg-zinc-800">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{detail.summary}</ReactMarkdown>
          </div>
        </section>
      ) : null}

      {detail.risks && detail.risks.length > 0 ? (
        <section className="border-t border-zinc-200 pt-4 dark:border-zinc-800">
          <RiskList risks={detail.risks} />
        </section>
      ) : null}

      {detail.suggestions && detail.suggestions.length > 0 ? (
        <section className="border-t border-zinc-200 pt-4 dark:border-zinc-800">
          <SuggestionList suggestions={detail.suggestions} />
        </section>
      ) : null}
    </article>
  );
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString("zh-CN", { hour12: false });
}
