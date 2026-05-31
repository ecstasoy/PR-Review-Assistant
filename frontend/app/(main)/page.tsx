"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { CapabilityCards } from "@/components/landing/CapabilityCards";
import { HeroBanner } from "@/components/landing/HeroBanner";
import { RecentReviewsList } from "@/components/landing/RecentReviewsList";
import { UrlInputCard } from "@/components/landing/UrlInputCard";
import { LoginGateBanner } from "@/components/landing/LoginGateBanner";
import { useMe } from "@/lib/auth";

// HomePage 落地页。提交 URL → 导航 /review/streaming?url=<encoded>，
// 流式渲染由 review 页面驱动。Landing 不再做 inline 渲染。
// 未登录用户看到 LoginGateBanner 替代 UrlInputCard（防匿名滥用 LLM cost）
export default function HomePage() {
  const router = useRouter();
  const [url, setUrl] = useState("");
  const { me, loading } = useMe();

  function start(target: string) {
    const encoded = encodeURIComponent(target.trim());
    router.push(`/review/streaming?url=${encoded}`);
  }

  return (
    <section className="mx-auto -mt-8 max-w-[720px] pt-[clamp(40px,9vh,96px)] pb-16">
      <HeroBanner />
      {loading ? (
        <div className="h-[180px] animate-pulse rounded-lg border border-border bg-surface-2" />
      ) : me?.authenticated ? (
        <UrlInputCard value={url} onChange={setUrl} onSubmit={start} />
      ) : (
        <LoginGateBanner />
      )}
      <CapabilityCards />
      <RecentReviewsList />
    </section>
  );
}
