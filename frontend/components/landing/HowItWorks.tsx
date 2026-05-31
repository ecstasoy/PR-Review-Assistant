import Link from "next/link";
import { Cog, ListChecks } from "lucide-react";

// HowItWorks 替代原 3 张 CapabilityCards；放最近评审之下
// 单列：左原理 / 右使用指引；窄屏自动堆叠
// 文案目标：让访客 30 秒看懂"做啥 + 怎么用"，避免读到一半放弃
export function HowItWorks() {
  return (
    <section className="mt-12 grid grid-cols-1 gap-5 md:grid-cols-2">
      <div className="rounded-lg border border-border bg-surface p-5">
        <div className="mb-3 flex items-center gap-2">
          <span className="inline-flex h-7 w-7 items-center justify-center rounded-md bg-surface-2 text-accent">
            <Cog className="h-3.5 w-3.5" />
          </span>
          <h2 className="text-sm font-semibold">工作原理</h2>
        </div>
        <p className="mb-3 text-[13px] leading-[1.7] text-text-2">
          拉 PR diff → RAG 在仓库里找相关跨文件代码 → 三轮并行 LLM 跑
          <strong className="font-medium text-text">变更总结 / 风险识别 / 行内建议</strong>
          。
        </p>
        <p className="text-[13px] leading-[1.7] text-text-2">
          建议按 GitHub <code className="rounded bg-surface-2 px-1 py-0.5 font-mono text-[11px]">suggestion</code>
          {" "}块格式打包，PR author 看到自带的「Apply」按钮可一键 commit。
        </p>
      </div>

      <div className="rounded-lg border border-border bg-surface p-5">
        <div className="mb-3 flex items-center gap-2">
          <span className="inline-flex h-7 w-7 items-center justify-center rounded-md bg-surface-2 text-accent">
            <ListChecks className="h-3.5 w-3.5" />
          </span>
          <h2 className="text-sm font-semibold">三步上手</h2>
        </div>
        <ol className="m-0 list-none space-y-2 p-0 text-[13px] leading-[1.6] text-text-2">
          <li className="flex gap-2">
            <span className="shrink-0 font-mono text-faint">1.</span>
            <span>
              顶栏「GitHub 登录」<span className="text-faint">（不要求 write 权限）</span>
            </span>
          </li>
          <li className="flex gap-2">
            <span className="shrink-0 font-mono text-faint">2.</span>
            <span>
              粘任意 PR 链接 <span className="text-faint">（私有 PR 需要装 LGTM App 到 repo）</span>
            </span>
          </li>
          <li className="flex gap-2">
            <span className="shrink-0 font-mono text-faint">3.</span>
            <span>
              看摘要 → 满意就点 <strong className="font-medium text-text">💬 评论到 PR</strong>{" "}
              或 <strong className="font-medium text-text">✅ 直接提交</strong>
            </span>
          </li>
        </ol>
        <p className="mt-4 border-t border-border pt-3 text-[12px] leading-[1.6] text-muted">
          想让 LGTM 自动评 repo 的所有 PR：
          <Link href="/" className="text-accent underline hover:opacity-80">
            装 LGTM App
          </Link>{" "}
          后 PR 一开 / push 新 commit 30s 内 bot 自动评；评论 <code className="rounded bg-surface-2 px-1 py-0.5 font-mono text-[11px]">/lgtm review</code> 可手动重触。
        </p>
      </div>
    </section>
  );
}
