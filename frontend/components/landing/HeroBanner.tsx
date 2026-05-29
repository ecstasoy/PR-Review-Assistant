// HeroBanner 资格 pill + h1 + 段落副标；纯 SSR 静态内容
// 标题字号用 clamp 跟随视口缩放（对齐 design 原型）

export function HeroBanner() {
  return (
    <header>
      <span className="mb-6 inline-flex items-center gap-[7px] rounded-full border border-border bg-surface px-3 py-1 font-mono text-xs text-muted">
        <span className="inline-block h-1.5 w-1.5 rounded-full bg-ok" aria-hidden />
        reviewer 视角 · 任意公开 PR 即用 · 无需仓库权限
      </span>
      <h1 className="m-0 mb-3.5 font-semibold leading-[1.1] tracking-[-0.02em] text-[clamp(28px,4.4vw,44px)]">
        粘一个 GitHub PR 链接，
        <br />
        <span className="text-muted">30 秒拿到结构化评审。</span>
      </h1>
      <p className="m-0 mb-7 max-w-[540px] text-base leading-[1.6] text-text-2">
        自动拉取 diff、扩展上下文，并行生成
        <strong className="font-semibold text-text">变更总结</strong>、
        <strong className="font-semibold text-text">风险识别</strong>与
        <strong className="font-semibold text-text">行内建议</strong>。结果可直接照搬进 GitHub 评论。
      </p>
    </header>
  );
}
