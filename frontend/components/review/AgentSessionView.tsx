"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import {
  AlertTriangle,
  AlignLeft,
  Check,
  ExternalLink,
  GitBranch,
  GitPullRequest,
  History as HistoryIcon,
  Send,
  Sparkle,
} from "lucide-react";

import type { File, PrMeta, Risk, Suggestion } from "@/lib/types";
import { streamSteer } from "@/lib/sse";
import { cn } from "@/lib/utils";
import { FileStatusBadge } from "@/components/ui/file-status-badge";
import { SeverityBadge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";

type StepStatus = "pending" | "running" | "done";

interface Props {
  pr: PrMeta;
  files: File[];
  risks: Risk[];
  suggestions: Suggestion[];
  summary: string;
  // 流式状态机：用于推断每步是 pending / running / done
  hasFiles: boolean;
  risksDone: boolean;
  suggestionsDone: boolean;
  streaming: boolean;
  // Steer：cached 模式才有 reviewId；streaming 模式 undefined，SteerComposer 禁用
  reviewId?: string;
  onSteeredRisks?: (risks: Risk[]) => void;
  onSteeredSuggestions?: (suggestions: Suggestion[]) => void;
}

// AgentSessionView 会话视图：把评审流程显示为 5 步 agent 时间线。
// 严格对齐 design 原型 AgentSession.jsx：parse → fetch → context → llm → cache。
// 状态由父组件传入的 SSE 状态机推断；cached 模式所有步骤 done。
export function AgentSessionView({
  pr,
  files,
  risks,
  suggestions,
  summary,
  hasFiles,
  risksDone,
  suggestionsDone,
  streaming,
  reviewId,
  onSteeredRisks,
  onSteeredSuggestions,
}: Props) {
  const scrollRef = useRef<HTMLDivElement>(null);

  // 5 步状态从 SSE 派生：
  //  - parse: pr 来即 done（顶层调用方保证 pr 不为 null 才渲染此组件）
  //  - fetch: hasFiles → done；否则 streaming 期间 running
  //  - context: files 来后立刻 running，summary 出现即 done（context 在 stages 启动前完成）
  //  - llm: summary 来 → running；suggestionsDone 后 → done
  //  - cache: 流结束（!streaming）且未报错 → done；否则 pending
  const statuses = useMemo<StepStatus[]>(() => {
    const hasSummary = summary.length > 0;
    return [
      "done", // parse 总是已完成（页面渲染时一定有 pr）
      hasFiles ? "done" : streaming ? "running" : "pending",
      hasSummary ? "done" : hasFiles ? "running" : "pending",
      suggestionsDone ? "done" : hasSummary ? "running" : "pending",
      !streaming && suggestionsDone ? "done" : "pending",
    ];
  }, [streaming, hasFiles, summary, risksDone, suggestionsDone]);

  const finished = !streaming && suggestionsDone;

  // 滚到底，让流式新增的步骤始终可见
  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [statuses, finished]);

  // 估算 token 预算（v1 用 patch 字节数粗算；后续 PR 接 BudgetReport SSE 帧后换真实值）
  const budget = useMemo(() => estimateBudget(pr, files), [pr, files]);

  return (
    <div className="flex h-full min-w-0 flex-col">
      <div ref={scrollRef} className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-[860px] px-6 pb-6 pt-5">
          <header className="mb-5 flex items-center gap-2.5">
            <span className="inline-flex h-[30px] w-[30px] items-center justify-center rounded-[9px] bg-accent text-accent-fg">
              <Sparkle className="h-[17px] w-[17px]" fill="currentColor" />
            </span>
            <div className="min-w-0">
              <div className="text-base font-semibold">
                评审 {pr.owner}/{pr.repo}#{pr.pr}
              </div>
              <div className="font-mono text-xs text-muted">
                reviewer 视角 · 任意公开 PR · 无需仓库权限
              </div>
            </div>
          </header>

          <Step
            icon={<GitBranch className="h-3.5 w-3.5" />}
            title="解析 PR URL"
            status={statuses[0]}
            meta={`${pr.owner}/${pr.repo}#${pr.pr}`}
          >
            <ParseChips pr={pr} />
          </Step>
          <Step
            icon={<GitPullRequest className="h-3.5 w-3.5" />}
            title="拉取 PR meta + diff"
            status={statuses[1]}
            meta={
              pr.stats
                ? `${pr.stats.files} files · +${pr.stats.additions} −${pr.stats.deletions}`
                : `${files.length} files`
            }
          >
            <FetchDetail files={files} pr={pr} />
          </Step>
          <Step
            icon={<AlignLeft className="h-3.5 w-3.5" />}
            title="构建三层上下文"
            status={statuses[2]}
            meta={`${(budget.total / 1000).toFixed(1)}K tok`}
          >
            <ContextDetail budget={budget} />
          </Step>
          <Step
            icon={<Sparkle className="h-3.5 w-3.5" />}
            title="并行调用 LLM"
            status={statuses[3]}
            meta={summary ? `${risks.length} 风险 · ${suggestions.length} 建议` : ""}
          >
            <LlmDetail
              summary={summary}
              risks={risks}
              suggestions={suggestions}
              hasSummary={summary.length > 0}
              risksDone={risksDone}
              suggestionsDone={suggestionsDone}
              streaming={streaming}
            />
          </Step>
          <Step
            icon={<HistoryIcon className="h-3.5 w-3.5" />}
            title="写入缓存"
            status={statuses[4]}
            meta={statuses[4] === "done" ? "SQLite" : ""}
            isLast
          >
            <CacheDetail pr={pr} />
          </Step>

          {finished ? (
            <FinalCard pr={pr} risks={risks} suggestions={suggestions} />
          ) : (
            <div className="flex items-center gap-2 pl-[38px] text-sm text-muted">
              <span className="flex gap-1">
                {[0, 1, 2].map((i) => (
                  <span
                    key={i}
                    className="inline-block h-1.5 w-1.5 rounded-full bg-muted"
                    style={{ animation: `pulse-dot 1s ${i * 0.18}s infinite` }}
                  />
                ))}
              </span>
              Agent 正在工作…
            </div>
          )}
        </div>
      </div>
      <SteerComposer
        pr={pr}
        reviewId={reviewId}
        onSteeredRisks={onSteeredRisks}
        onSteeredSuggestions={onSteeredSuggestions}
      />
    </div>
  );
}

// ─────────────── Step shell ───────────────

function Step({
  icon,
  title,
  status,
  meta,
  isLast,
  children,
}: {
  icon: React.ReactNode;
  title: string;
  status: StepStatus;
  meta?: string;
  isLast?: boolean;
  children?: React.ReactNode;
}) {
  const isDone = status === "done";
  const isRunning = status === "running";
  const isPending = status === "pending";
  return (
    <div className="grid grid-cols-[26px_1fr] gap-3">
      <div className="flex flex-col items-center">
        <span
          className={cn(
            "inline-flex h-[26px] w-[26px] shrink-0 items-center justify-center rounded-md border",
            isPending
              ? "border-border bg-surface-2 text-faint"
              : isRunning
                ? "border-border-strong bg-surface text-accent"
                : "border-border-strong bg-surface text-ok",
          )}
        >
          {isRunning ? <Spinner size="xs" className="text-accent" /> : icon}
        </span>
        {!isLast ? (
          <span className="my-1 min-h-3 w-[1.5px] flex-1 bg-border" />
        ) : null}
      </div>
      <div className={cn("min-w-0", isLast ? "pb-0" : "pb-4")}>
        <div className="flex min-h-[26px] items-center gap-2">
          <span
            className={cn(
              "whitespace-nowrap text-sm font-semibold",
              isPending ? "text-muted" : "text-text",
            )}
          >
            {title}
          </span>
          {isRunning ? (
            <span className="whitespace-nowrap font-mono text-[10px] text-accent">运行中…</span>
          ) : null}
          {meta && isDone ? (
            <span className="ml-auto whitespace-nowrap font-mono text-[10.5px] text-faint">
              {meta}
            </span>
          ) : null}
        </div>
        {!isPending && children ? <div className="mt-2">{children}</div> : null}
      </div>
    </div>
  );
}

function ToolCard({
  label,
  children,
}: {
  label?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="overflow-hidden rounded-md border border-border bg-surface-2">
      {label ? (
        <div className="flex items-center gap-1.5 border-b border-border px-2.5 py-1.5 font-mono text-[10.5px] text-muted">
          {label}
        </div>
      ) : null}
      <div className="p-2.5">{children}</div>
    </div>
  );
}

// ─────────────── Step detail renderers ───────────────

function ParseChips({ pr }: { pr: PrMeta }) {
  const entries: Array<[string, string]> = [
    ["owner", pr.owner],
    ["repo", pr.repo],
    ["pr", `#${pr.pr}`],
    ["head", pr.head_sha.slice(0, 11)],
  ];
  return (
    <div className="flex flex-wrap gap-1.5">
      {entries.map(([k, v]) => (
        <code
          key={k}
          className="rounded-md border border-border bg-surface-2 px-2 py-0.5 font-mono text-[10.5px]"
        >
          <span className="text-faint">{k}</span> {v}
        </code>
      ))}
    </div>
  );
}

function FetchDetail({ files, pr }: { files: File[]; pr: PrMeta }) {
  if (files.length === 0) {
    return <p className="text-xs text-muted">（无文件改动）</p>;
  }
  return (
    <ToolCard
      label={
        <>
          <GitPullRequest className="h-3 w-3" /> GET /repos/{pr.owner}/{pr.repo}/pulls/{pr.pr}/files
        </>
      }
    >
      <div className="flex flex-col gap-1">
        {files.slice(0, 10).map((f) => (
          <div key={f.path} className="flex items-center gap-2">
            <FileStatusBadge status={f.status} />
            <code className="min-w-0 flex-1 truncate font-mono text-xs">{f.path}</code>
            <span className="flex shrink-0 gap-1 font-mono text-[10px]">
              <span className="text-ok">+{f.additions}</span>
              <span className="text-high">−{f.deletions}</span>
            </span>
          </div>
        ))}
        {files.length > 10 ? (
          <p className="pt-1 text-[10px] text-faint">…还有 {files.length - 10} 个文件</p>
        ) : null}
      </div>
    </ToolCard>
  );
}

interface BudgetItem {
  layer: "L1" | "L2" | "L3";
  label: string;
  tok: number;
  className: string;
}

function estimateBudget(_pr: PrMeta, files: File[]): { items: BudgetItem[]; total: number } {
  // 粗估：L2 = patch 字节数 / 3（与后端 estimateTokens 同算法）；L1/L3 用固定值
  let patchChars = 0;
  for (const f of files) patchChars += f.patch?.length ?? 0;
  const l2 = Math.max(200, Math.round(patchChars / 3));
  const l1 = 400 + files.length * 30; // meta + per-file summary
  const l3 = Math.min(1600, Math.round((l1 + l2) / 12)); // 约 4:5:1 中 L3 那份
  const items: BudgetItem[] = [
    { layer: "L1", label: "diff hunk + PR meta", tok: l1, className: "bg-info" },
    { layer: "L2", label: "变更文件 / 受影响函数", tok: l2, className: "bg-ok" },
    { layer: "L3", label: "项目约定（README/CLAUDE.md）", tok: l3, className: "bg-med" },
  ];
  return { items, total: l1 + l2 + l3 };
}

function ContextDetail({ budget }: { budget: { items: BudgetItem[]; total: number } }) {
  return (
    <ToolCard
      label={
        <>token 预算 · L1:L2:L3 ≈ 4:5:1 · 合计 {(budget.total / 1000).toFixed(1)}K</>
      }
    >
      <div className="mb-2.5 flex h-2 overflow-hidden rounded-full">
        {budget.items.map((b) => (
          <span
            key={b.layer}
            className={b.className}
            style={{ width: `${(b.tok / budget.total) * 100}%` }}
          />
        ))}
      </div>
      <div className="flex flex-col gap-1.5">
        {budget.items.map((b) => (
          <div key={b.layer} className="flex items-center gap-2">
            <span className={cn("h-2 w-2 shrink-0 rounded-sm", b.className)} />
            <code className="w-5 font-mono text-[10.5px] font-semibold">{b.layer}</code>
            <span className="min-w-0 flex-1 truncate text-xs text-text-2">{b.label}</span>
            <span className="font-mono text-[10.5px] text-muted">
              {(b.tok / 1000).toFixed(2)}K
            </span>
          </div>
        ))}
      </div>
      <div className="mt-2.5 flex flex-wrap gap-1.5">
        {["README.md", "CONTRIBUTING.md", "CLAUDE.md"].map((f) => (
          <code
            key={f}
            className="rounded border border-med-bd bg-med-bg px-1.5 py-0.5 font-mono text-[10px] text-med"
          >
            {f}
          </code>
        ))}
      </div>
    </ToolCard>
  );
}

function LlmDetail({
  summary,
  risks,
  suggestions,
  hasSummary,
  risksDone,
  suggestionsDone,
  streaming,
}: {
  summary: string;
  risks: Risk[];
  suggestions: Suggestion[];
  hasSummary: boolean;
  risksDone: boolean;
  suggestionsDone: boolean;
  streaming: boolean;
}) {
  // 三泳道：summary / risks / suggestions
  // 状态：running 期间未完成 → "running"；完成 → "done"；上游未启 → "pending"
  const stage = (done: boolean, started: boolean): StepStatus =>
    done ? "done" : started ? "running" : "pending";
  const summaryStage: StepStatus =
    summary && (risksDone || !streaming) ? "done" : hasSummary ? "running" : "pending";
  return (
    <ToolCard
      label={
        <>
          <Sparkle className="h-3 w-3" fill="currentColor" /> fan-out · 3 阶段并行（不共享上下文，降低相互污染）
        </>
      }
    >
      <Lane
        icon={<AlignLeft className="h-3 w-3" />}
        label="总结"
        status={summaryStage}
        result={summaryStage === "done" ? "已到达" : ""}
      />
      <Lane
        icon={<AlertTriangle className="h-3 w-3" />}
        label="风险识别"
        status={stage(risksDone, hasSummary)}
        result={risksDone ? `${risks.length} 项` : ""}
      />
      <Lane
        icon={<Sparkle className="h-3 w-3" />}
        label="行内建议"
        status={stage(suggestionsDone, risksDone)}
        result={suggestionsDone ? `${suggestions.length} 条` : ""}
      />
      {summary ? (
        <div className="mt-2 border-t border-dashed border-border pt-2.5 text-sm leading-relaxed text-text-2 line-clamp-6">
          {summary}
        </div>
      ) : null}
      {risksDone && risks.length > 0 ? (
        <div className="mt-1 flex flex-wrap gap-1.5">
          {risks.slice(0, 4).map((r, i) => (
            <span
              key={i}
              className="inline-flex items-center gap-1.5 rounded-full border border-border bg-surface px-2 py-0.5"
            >
              <SeverityBadge severity={r.severity} className="px-1.5 py-0 text-[10px]">
                {r.severity}
              </SeverityBadge>
              <code className="font-mono text-[10px] text-text-2">
                {r.file.split("/").pop()}
                {r.line ? `:${r.line}` : ""}
              </code>
            </span>
          ))}
        </div>
      ) : null}
    </ToolCard>
  );
}

function Lane({
  icon,
  label,
  status,
  result,
}: {
  icon: React.ReactNode;
  label: string;
  status: StepStatus;
  result?: string;
}) {
  return (
    <div className="flex items-center gap-2 py-[7px]">
      <span className="flex w-[18px] shrink-0 justify-center">
        {status === "running" ? (
          <Spinner size="xs" className="text-accent" />
        ) : status === "done" ? (
          <Check className="h-3.5 w-3.5 text-ok" strokeWidth={2.4} />
        ) : (
          <span className="inline-block h-2.5 w-2.5 rounded-full border-[1.5px] border-border-strong" />
        )}
      </span>
      <span className="text-muted">{icon}</span>
      <span
        className={cn(
          "text-sm font-medium",
          status === "pending" ? "text-muted" : "text-text",
        )}
      >
        {label}
      </span>
      <span className="ml-auto font-mono text-[10.5px] text-faint">
        {status === "done" ? result : status === "running" ? "流式生成…" : "排队"}
      </span>
    </div>
  );
}

function CacheDetail({ pr }: { pr: PrMeta }) {
  return (
    <ToolCard label="SQLite · 写入缓存">
      <code className="font-mono text-xs text-text-2">
        key = <span className="text-info">{pr.owner}/{pr.repo}</span>:
        <span className="text-ok">{pr.pr}</span>:
        <span className="text-med">{pr.head_sha.slice(0, 11)}</span>
      </code>
      <div className="mt-1.5 text-xs text-muted">head_sha 不变时下次秒回。</div>
    </ToolCard>
  );
}

function FinalCard({
  pr: _pr,
  risks,
  suggestions,
}: {
  pr: PrMeta;
  risks: Risk[];
  suggestions: Suggestion[];
}) {
  return (
    <div className="pl-[38px]">
      <div className="mb-3 flex items-center gap-2 rounded-lg border border-border bg-surface px-3.5 py-2.5">
        <Check className="h-4 w-4 text-ok" strokeWidth={2.4} />
        <span className="text-sm font-semibold">评审完成</span>
        <span className="text-xs text-muted">
          总结 + {risks.length} 风险 + {suggestions.length} 建议
        </span>
      </div>
      <div className="flex flex-wrap items-center gap-2.5 font-mono text-[10.5px] text-faint">
        <span className="inline-flex items-center gap-1">
          <Sparkle className="h-[11px] w-[11px] text-accent" fill="currentColor" />
          DeepSeek · deepseek-chat <span className="text-med">(mock)</span>
        </span>
        <span>· 上下文已含 README / CLAUDE.md</span>
      </div>
    </div>
  );
}

// SteerComposer 底部「引导」输入条；接 POST /api/review/:id/steer。
// reviewId 缺失（streaming 模式）→ 禁用 + 提示「等流式完成后可引导」
function SteerComposer({
  pr,
  reviewId,
  onSteeredRisks,
  onSteeredSuggestions,
}: {
  pr: PrMeta;
  reviewId?: string;
  onSteeredRisks?: (risks: Risk[]) => void;
  onSteeredSuggestions?: (suggestions: Suggestion[]) => void;
}) {
  const [text, setText] = useState("");
  const [stage, setStage] = useState<"risks" | "suggestions">("risks");
  const [inFlight, setInFlight] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const githubURL = `https://github.com/${pr.owner}/${pr.repo}/pull/${pr.pr}`;
  const enabled = !!reviewId && !inFlight;

  async function send() {
    const v = text.trim();
    if (!v || !reviewId || inFlight) return;
    setInFlight(true);
    setError(null);
    let stageFailed = false;
    try {
      await streamSteer(reviewId, v, stage, {
        onSteeredRisks: (r) => onSteeredRisks?.(r),
        onSteeredSuggestions: (s) => onSteeredSuggestions?.(s),
        onStageError: (_s, msg) => {
          stageFailed = true;
          setError(msg);
        },
      });
      if (!stageFailed) setText("");
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setInFlight(false);
    }
  }

  return (
    <div className="border-t border-border bg-surface px-6 py-3">
      <div className="mx-auto max-w-[860px]">
        <div className="mb-2 flex items-center gap-2">
          <span className="inline-flex items-center gap-1.5 rounded-md border border-border bg-surface-2 px-2.5 py-1 font-mono text-xs text-info">
            <GitBranch className="h-3 w-3" />
            {pr.head_ref || "head"}
            {pr.stats ? (
              <>
                <span className="text-ok">+{pr.stats.additions}</span>
                <span className="text-high">−{pr.stats.deletions}</span>
              </>
            ) : null}
          </span>
          <div className="flex gap-[3px] rounded-md border border-border bg-surface-2 p-[2px]">
            {(["risks", "suggestions"] as const).map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => setStage(s)}
                className={cn(
                  "rounded-sm px-2 py-[3px] font-mono text-[10.5px] transition-colors",
                  stage === s ? "bg-surface text-text" : "text-muted hover:text-text",
                )}
              >
                {s === "risks" ? "重评风险" : "重出建议"}
              </button>
            ))}
          </div>
          <a
            href={githubURL}
            target="_blank"
            rel="noreferrer"
            className="ml-auto inline-flex h-7 items-center gap-1 rounded-md border border-border-strong bg-surface px-2.5 text-xs text-text-2 hover:bg-surface-hover hover:text-text"
          >
            <ExternalLink className="h-3 w-3" />
            查看 PR
          </a>
        </div>
        <div
          className={cn(
            "flex items-end gap-2 rounded-lg border bg-surface p-1.5 shadow-sm",
            enabled ? "border-border-strong" : "border-border",
          )}
        >
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            disabled={!enabled}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                send();
              }
            }}
            rows={1}
            placeholder={
              reviewId
                ? `引导 agent ${stage === "risks" ? "重评风险" : "重出建议"}（例：重点看并发安全 / 忽略 style 类问题）…`
                : "流式评审完成后可在此引导 agent"
            }
            className="max-h-[120px] min-w-0 flex-1 resize-none border-none bg-transparent px-1.5 py-1.5 text-sm leading-snug text-text outline-none placeholder:text-faint disabled:cursor-not-allowed"
          />
          <button
            type="button"
            onClick={send}
            disabled={!enabled || !text.trim()}
            className="inline-flex h-[30px] items-center rounded-md bg-accent px-2.5 text-accent-fg hover:opacity-90 disabled:opacity-50"
          >
            {inFlight ? <Spinner size="xs" className="text-accent-fg" /> : <Send className="h-3.5 w-3.5" />}
          </button>
        </div>
        {error ? (
          <p className="mt-1 text-[10.5px] text-high">引导失败：{error}</p>
        ) : null}
      </div>
    </div>
  );
}
