interface Props {
  reviewId: string;
}

// PR #5 用 react-markdown 渲染；PR #12 接 SSE 流式增量
export function SummaryCard({ reviewId }: Props) {
  return (
    <article className="rounded-lg border border-zinc-200 p-5 dark:border-zinc-800">
      <h2 className="mb-2 text-lg font-medium">变更总结</h2>
      <p className="text-sm text-zinc-500">Pending wiring (review id: {reviewId}).</p>
    </article>
  );
}
