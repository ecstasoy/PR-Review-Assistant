"use client";

import { useEffect } from "react";
import Link from "next/link";
import { AlertTriangle } from "lucide-react";

// app/error.tsx App Router 路由级错误边界。
// 任何 client component 未捕获的 throw / promise reject 落到这里；
// 不取代 review 页内的 stage 级 error banner——那里 ErrorBoundary 之上有自定义处理。
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // eslint-disable-next-line no-console
    console.error("Unhandled UI error", error);
  }, [error]);

  return (
    <section className="mx-auto max-w-[480px] px-6 py-16 text-center">
      <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-full bg-high-bg text-high">
        <AlertTriangle className="h-6 w-6" />
      </div>
      <h1 className="text-lg font-semibold">页面出错了</h1>
      <p className="mt-2 text-sm text-muted">
        {error.message || "未知错误"}
        {error.digest ? (
          <span className="ml-2 font-mono text-faint">[{error.digest}]</span>
        ) : null}
      </p>
      <div className="mt-5 flex justify-center gap-2.5">
        <button
          type="button"
          onClick={() => reset()}
          className="rounded-md bg-accent px-3.5 py-1.5 text-sm font-medium text-accent-fg hover:opacity-90"
        >
          重试
        </button>
        <Link
          href="/"
          className="rounded-md border border-border-strong bg-surface px-3.5 py-1.5 text-sm text-text-2 hover:bg-surface-hover hover:text-text"
        >
          回到落地页
        </Link>
      </div>
    </section>
  );
}
