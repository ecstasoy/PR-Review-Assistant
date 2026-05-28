// PR #14 调 GET /api/reviews 渲染分页列表
export function HistoryList() {
  return (
    <p className="text-sm text-zinc-500">
      尚无评审记录。提交一个 PR 后会出现在这里。
    </p>
  );
}
