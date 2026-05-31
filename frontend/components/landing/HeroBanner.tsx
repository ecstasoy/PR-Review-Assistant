// HeroBanner h1 + 一句话副标；不含 pill / 不含三块 capability
// 标题字号 clamp 跟随视口；纯 SSR 静态内容

export function HeroBanner() {
  return (
    <header>
      <h1 className="m-0 mb-3.5 font-semibold leading-[1.1] tracking-[-0.02em] text-[clamp(28px,4.4vw,44px)]">
        Looks good to me?
        <br />
        <span className="text-muted">Looks good to you!</span>
      </h1>
      <p className="m-0 mb-7 max-w-[540px] text-base leading-[1.6] text-text-2">
        粘个 PR 链接，<strong className="font-semibold text-text">LGTM</strong> 三十秒内给你
        <strong className="font-semibold text-text">总结 / 风险 / 行内建议</strong>，
        可一键发到原 PR。
      </p>
    </header>
  );
}
