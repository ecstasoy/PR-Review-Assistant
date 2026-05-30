// app/review/[id]/loading.tsx Suspense fallback 骨架屏。
// streaming 模式首字节前 + cached 模式 fetch 中显示；闪烁过渡比直接旋转 spinner 更友好。
export default function ReviewLoading() {
  return (
    <div className="flex h-screen flex-col bg-bg">
      <header className="flex h-12 items-center gap-3 border-b border-border bg-surface px-4">
        <span className="h-5 w-5 animate-pulse rounded-full bg-surface-2" />
        <span className="h-4 w-48 animate-pulse rounded bg-surface-2" />
        <span className="ml-auto h-7 w-32 animate-pulse rounded-md bg-surface-2" />
      </header>
      <div className="flex min-h-0 flex-1">
        <aside className="hidden w-[256px] flex-col gap-3 border-r border-border bg-surface p-4 md:flex">
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="flex flex-col gap-1.5">
              <span className="h-3 w-3/4 animate-pulse rounded bg-surface-2" />
              <span className="h-3 w-1/2 animate-pulse rounded bg-surface-2" />
            </div>
          ))}
        </aside>
        <main className="min-w-0 flex-1 overflow-y-auto px-6 py-5">
          <div className="mx-auto flex max-w-[1080px] flex-col gap-4">
            <div className="h-36 animate-pulse rounded-lg border border-border bg-surface" />
            <div className="h-72 animate-pulse rounded-lg border border-border bg-surface" />
          </div>
        </main>
      </div>
    </div>
  );
}
