import Link from "next/link";
import { Compass } from "lucide-react";

// app/not-found.tsx 全局 404。
// 触发：访问不存在的路由 / 任何 server component throw notFound()
export default function NotFound() {
  return (
    <section className="mx-auto max-w-[480px] px-6 py-16 text-center">
      <div className="mb-4 inline-flex h-12 w-12 items-center justify-center rounded-full bg-surface-2 text-muted">
        <Compass className="h-6 w-6" />
      </div>
      <h1 className="text-lg font-semibold">页面不存在</h1>
      <p className="mt-2 text-sm text-muted">
        URL 可能拼错了，或者评审记录已被删除。
      </p>
      <div className="mt-5 flex justify-center gap-2.5">
        <Link
          href="/"
          className="rounded-md bg-accent px-3.5 py-1.5 text-sm font-medium text-accent-fg hover:opacity-90"
        >
          回到落地页
        </Link>
        <Link
          href="/history"
          className="rounded-md border border-border-strong bg-surface px-3.5 py-1.5 text-sm text-text-2 hover:bg-surface-hover hover:text-text"
        >
          浏览历史
        </Link>
      </div>
    </section>
  );
}
