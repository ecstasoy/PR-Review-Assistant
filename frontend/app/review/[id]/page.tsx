"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";

import { getReview } from "@/lib/api";
import type { ReviewDetail } from "@/lib/types";
import { ReviewTopBar } from "@/components/review/ReviewTopBar";
import { SummaryCard } from "@/components/review/SummaryCard";
import { RisksList } from "@/components/review/RisksList";
import { SuggestionList } from "@/components/SuggestionList";
import { Spinner } from "@/components/ui/spinner";

interface PageProps {
  params: Promise<{ id: string }>;
}

// ReviewDetail 在 API 层扩了 ci 字段（A3 持久化），lib/types 还没同步，临时本地扩
interface DetailWithCI extends ReviewDetail {
  ci?: string;
}

// /review/[id] 评审结果页（cached-only）。
// design 的完整顶栏（view switch / stage chips / 追问 dock）+ 左侧栏 + Diff 视图留下个 PR。
// 本页只渲染报告视图：PR 头部 + 变更总结 + 风险识别 +（暂时复用旧 SuggestionList）行内建议。
export default function ReviewDetailPage({ params }: PageProps) {
  const { id } = use(params);
  const [detail, setDetail] = useState<DetailWithCI | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    getReview(id)
      .then((d) => {
        if (!cancelled) setDetail(d as DetailWithCI);
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
        <Link href="/history" className="text-xs text-muted hover:text-text">
          ← 返回历史
        </Link>
        <p className="text-sm text-fail">加载失败：{error}</p>
      </section>
    );
  }
  if (!detail) {
    return (
      <p className="flex items-center gap-2 text-sm text-muted">
        <Spinner size="xs" /> 加载中…
      </p>
    );
  }

  return (
    <section className="space-y-3">
      <ReviewTopBar
        title={detail.title || `${detail.owner}/${detail.repo}#${detail.pr}`}
        owner={detail.owner}
        repo={detail.repo}
        pr={detail.pr}
        headSha={detail.head_sha}
        ci={detail.ci}
      />
      <SummaryCard summary={detail.summary} />
      {detail.risks && detail.risks.length > 0 ? <RisksList risks={detail.risks} /> : null}
      {detail.suggestions && detail.suggestions.length > 0 ? (
        <SuggestionList suggestions={detail.suggestions} />
      ) : null}
    </section>
  );
}
