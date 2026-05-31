"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { HeroBanner } from "@/components/landing/HeroBanner";
import { RecentReviewsList } from "@/components/landing/RecentReviewsList";
import { UrlInputCard } from "@/components/landing/UrlInputCard";
import { LoginGateBanner } from "@/components/landing/LoginGateBanner";
import { useMe } from "@/lib/auth";

// HomePage 落地页。提交 URL → 导航 /review/streaming?url=<encoded>，流式渲染由 review 页面驱动。
// 未登录：LoginGateBanner 替代 UrlInputCard + 隐藏「最近评审」列表（含他人评的 PR 内容，登录后才看）
export default function HomePage() {
  const router = useRouter();
  const [url, setUrl] = useState("");
  const { me, loading } = useMe();

  function start(target: string) {
    const encoded = encodeURIComponent(target.trim());
    router.push(`/review/streaming?url=${encoded}`);
  }

  const authenticated = !!me?.authenticated;

  return (
    <section className="mx-auto -mt-8 max-w-[720px] pt-[clamp(40px,9vh,96px)] pb-16">
      <HeroBanner />
      {loading ? (
        <div className="h-[180px] animate-pulse rounded-lg border border-border bg-surface-2" />
      ) : authenticated ? (
        <UrlInputCard value={url} onChange={setUrl} onSubmit={start} />
      ) : (
        <LoginGateBanner />
      )}
      {authenticated ? <RecentReviewsList /> : null}

      {/* 简介：评审任何 PR 不需要装 App；想直接发回 GitHub 才装 */}
      <p className="mt-10 max-w-[640px] text-sm leading-[1.7] text-text-2">
        登录后即可对任意 public PR 评审；如果想把 LGTM 给出的修改建议
        <strong className="font-medium text-text"> 一键发回 GitHub PR 评论 / 提交 commit</strong>
        ，需要给对应 repo 安装{" "}
        <a
          href="https://github.com/apps/lgtm-ai-reviewer"
          target="_blank"
          rel="noreferrer"
          className="text-accent underline hover:opacity-80"
        >
          LGTM App
        </a>
        （只读 diff + 写 PR 评论权限）。装完还能享受 webhook 自动评：开 PR / push 新 commit
        bot 30 秒内自动回贴评审。
      </p>
    </section>
  );
}
