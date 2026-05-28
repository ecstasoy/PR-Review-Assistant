import { ReviewForm } from "@/components/ReviewForm";

export default function Home() {
  return (
    <section className="space-y-6">
      <header>
        <h1 className="text-3xl font-semibold tracking-tight">
          AI PR Review 助手
        </h1>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          粘贴一个公开的 GitHub PR 链接，30 秒拿到总结、风险与行内建议。
        </p>
      </header>
      <ReviewForm />
    </section>
  );
}
