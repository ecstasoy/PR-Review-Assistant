"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

import { CapabilityCards } from "@/components/landing/CapabilityCards";
import { HeroBanner } from "@/components/landing/HeroBanner";
import { RecentReviewsList } from "@/components/landing/RecentReviewsList";
import { UrlInputCard } from "@/components/landing/UrlInputCard";

// HomePage 落地页。提交 URL → 导航 /review/streaming?url=<encoded>，
// 流式渲染由 review 页面驱动。Landing 不再做 inline 渲染。
export default function HomePage() {
  const router = useRouter();
  const [url, setUrl] = useState("");

  function start(target: string) {
    const encoded = encodeURIComponent(target.trim());
    router.push(`/review/streaming?url=${encoded}`);
  }

  return (
    <section className="mx-auto -mt-8 max-w-[720px] pt-[clamp(40px,9vh,96px)] pb-16">
      <HeroBanner />
      <UrlInputCard value={url} onChange={setUrl} onSubmit={start} />
      <CapabilityCards />
      <RecentReviewsList />
    </section>
  );
}
