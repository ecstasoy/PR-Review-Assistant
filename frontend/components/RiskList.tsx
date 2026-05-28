interface Props {
  reviewId: string;
}

// PR #10 按 severity 分组排序；low 默认折叠
export function RiskList({ reviewId }: Props) {
  return (
    <article className="rounded-lg border border-zinc-200 p-5 dark:border-zinc-800">
      <h2 className="mb-2 text-lg font-medium">风险识别</h2>
      <p className="text-sm text-zinc-500">Pending wiring (review id: {reviewId}).</p>
    </article>
  );
}
