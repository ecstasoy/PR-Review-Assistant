import { HistoryList } from "@/components/HistoryList";

export default function HistoryPage() {
  return (
    <section className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">历史评审</h1>
        <p className="mt-2 text-sm text-zinc-600 dark:text-zinc-400">
          最近评审过的 PR；点进去可查看缓存结果。
        </p>
      </header>
      <HistoryList />
    </section>
  );
}
