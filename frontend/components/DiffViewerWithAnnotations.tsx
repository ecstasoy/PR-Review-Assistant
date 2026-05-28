interface Props {
  reviewId: string;
}

// PR #10 左右分栏 diff + 行号锚定的 AI 建议气泡
export function DiffViewerWithAnnotations({ reviewId }: Props) {
  return (
    <article className="rounded-lg border border-zinc-200 p-5 dark:border-zinc-800">
      <h2 className="mb-2 text-lg font-medium">行内建议</h2>
      <p className="text-sm text-zinc-500">Pending wiring (review id: {reviewId}).</p>
    </article>
  );
}
