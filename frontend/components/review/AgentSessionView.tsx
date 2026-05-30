"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  AlertTriangle,
  AlignLeft,
  Check,
  ExternalLink,
  GitBranch,
  GitPullRequest,
  History as HistoryIcon,
  MessageSquare,
  Send,
  Sparkle,
  Wrench,
} from "lucide-react";

import type {
  AgentToolCall,
  BudgetReport,
  File,
  PrMeta,
  Risk,
  Suggestion,
} from "@/lib/types";
import { streamSteer, type SteerMode } from "@/lib/sse";
import { cn } from "@/lib/utils";
import { FileStatusBadge } from "@/components/ui/file-status-badge";
import { SeverityBadge } from "@/components/ui/badge";
import { Spinner } from "@/components/ui/spinner";

type StepStatus = "pending" | "running" | "done" | "error";

// SteerEntry 一次用户引导的记录；时间线追加到 5 步主流程之后
interface SteerEntry {
  id: string;
  text: string;
  stage: "risks" | "suggestions";
  mode: SteerMode; // stage 重跑 / agent 跑 ReAct loop
  status: "running" | "done" | "error";
  resultCount?: number;
  error?: string;
}

interface Props {
  pr: PrMeta;
  files: File[];
  risks: Risk[];
  suggestions: Suggestion[];
  summary: string;
  // budget 后端真值；null 时回退到 patch-bytes 粗估（流式 budget_report 帧到达前 / 旧缓存）
  budget: BudgetReport | null;
  // 流式状态机：用于推断每步是 pending / running / done
  hasFiles: boolean;
  risksDone: boolean;
  suggestionsDone: boolean;
  streaming: boolean;
  // Steer：cached 模式才有 reviewId；streaming 模式 undefined，SteerComposer 禁用
  reviewId?: string;
  onSteeredRisks?: (risks: Risk[]) => void;
  onSteeredSuggestions?: (suggestions: Suggestion[]) => void;
  // Agent loop tool 调用事件（A3 后端 emit tool_call_start/done）
  // 按 id 已合并 start/done，状态从 running → done/error；从父级 state 流下来
  toolEvents?: ToolEvent[];
}

// ToolEvent agent loop 单次工具调用的合并状态
export interface ToolEvent {
  id: string;
  name: string;
  arguments?: string;
  result?: string;
  status: "running" | "done" | "error";
}

// MergeToolEvent 给 page.tsx 复用：根据 tool_call_start / done 帧累积 ToolEvent[]
// 父组件用 reducer 累积，子组件只渲染。
export function mergeToolStart(prev: ToolEvent[], call: AgentToolCall): ToolEvent[] {
  // 同 id 二次 start 应该不会发生；防御性覆盖
  const idx = prev.findIndex((e) => e.id === call.id);
  const next: ToolEvent = {
    id: call.id,
    name: call.name,
    arguments: call.arguments,
    status: "running",
  };
  if (idx >= 0) {
    const copy = [...prev];
    copy[idx] = next;
    return copy;
  }
  return [...prev, next];
}

export function mergeToolDone(prev: ToolEvent[], call: AgentToolCall): ToolEvent[] {
  const idx = prev.findIndex((e) => e.id === call.id);
  if (idx < 0) {
    // 没见过 start 也兼容（理论不应发生）：直接当 done 插入
    return [
      ...prev,
      {
        id: call.id,
        name: call.name,
        result: call.result,
        status: call.result?.startsWith("error:") ? "error" : "done",
      },
    ];
  }
  const copy = [...prev];
  copy[idx] = {
    ...copy[idx],
    result: call.result,
    status: call.result?.startsWith("error:") ? "error" : "done",
  };
  return copy;
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
  budget,
  hasFiles,
  risksDone,
  suggestionsDone,
  streaming,
  reviewId,
  onSteeredRisks,
  onSteeredSuggestions,
  toolEvents,
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

  // steer 历史：每条引导一个步骤；状态机由 streamSteer 回调驱动
  const [steerHistory, setSteerHistory] = useState<SteerEntry[]>([]);
  const [steerInFlight, setSteerInFlight] = useState(false);
  const steerInFlightRef = useRef(false);

  const handleSteerSend = useCallback(
    async (text: string, stage: "risks" | "suggestions", mode: SteerMode) => {
      if (!reviewId || steerInFlightRef.current) return;
      steerInFlightRef.current = true;
      const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      setSteerHistory((prev) => [...prev, { id, text, stage, mode, status: "running" }]);
      setSteerInFlight(true);
      const markDone = (count: number) =>
        setSteerHistory((prev) =>
          prev.map((e) => (e.id === id ? { ...e, status: "done", resultCount: count } : e)),
        );
      const markError = (msg: string) =>
        setSteerHistory((prev) =>
          prev.map((e) => (e.id === id ? { ...e, status: "error", error: msg } : e)),
        );
      try {
        await streamSteer(
          reviewId,
          text,
          stage,
          {
            onSteeredRisks: (r) => {
              onSteeredRisks?.(r);
              if (mode === "stage" && stage === "risks") markDone(r.length);
            },
            onSteeredSuggestions: (s) => {
              onSteeredSuggestions?.(s);
              if (mode === "stage" && stage === "suggestions") markDone(s.length);
            },
            onStageError: (_s, msg) => markError(msg),
          },
          undefined,
          mode,
        );
        // 兜底：极端情况下流结束仍未拿到 steered 帧，标 done 不报错
        // agent 模式不期待 steered_* 帧，工具调用结果走 toolEvents 时间线
        setSteerHistory((prev) =>
          prev.map((e) =>
            e.id === id && e.status === "running" ? { ...e, status: "done", resultCount: 0 } : e,
          ),
        );
      } catch (e) {
        markError(e instanceof Error ? e.message : String(e));
      } finally {
        steerInFlightRef.current = false;
        setSteerInFlight(false);
      }
    },
    [reviewId, onSteeredRisks, onSteeredSuggestions],
  );

  const finished = !streaming && suggestionsDone;

  // 滚到底，让流式新增的步骤始终可见
  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [statuses, finished, (toolEvents ?? []).length]);

  // budget 优先用后端真值（SSE budget_report / cached detail）；缺失时回退到 patch-bytes 粗估
  const budgetView = useMemo(
    () => (budget ? fromBudgetReport(budget) : estimateBudget(pr, files)),
    [budget, pr, files],
  );

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
            meta={`${(budgetView.total / 1000).toFixed(1)}K tok${budgetView.estimated ? "（粗估）" : ""}`}
          >
            <ContextDetail budget={budgetView} />
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
          {(toolEvents ?? []).map((evt) => (
            <Step
              key={evt.id}
              icon={<Wrench className="h-3.5 w-3.5" />}
              title={`调用 ${evt.name}`}
              status={evt.status}
              meta={toolMeta(evt)}
            >
              <ToolEventDetail evt={evt} />
            </Step>
          ))}

          <Step
            icon={<HistoryIcon className="h-3.5 w-3.5" />}
            title="写入缓存"
            status={statuses[4]}
            meta={statuses[4] === "done" ? "SQLite" : ""}
            isLast={steerHistory.length === 0}
          >
            <CacheDetail pr={pr} />
          </Step>

          {steerHistory.map((entry, i) => (
            <Step
              key={entry.id}
              icon={<MessageSquare className="h-3.5 w-3.5" />}
              title="用户引导"
              status={entry.status}
              meta={steerMeta(entry)}
              isLast={i === steerHistory.length - 1}
            >
              <SteerDetail entry={entry} />
            </Step>
          ))}

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
        disabled={!reviewId}
        inFlight={steerInFlight}
        onSend={handleSteerSend}
      />
    </div>
  );
}

function steerMeta(entry: SteerEntry): string {
  const label = entry.stage === "risks" ? "重评风险" : "重出建议";
  if (entry.status === "running") return `${label} · 运行中`;
  if (entry.status === "error") return `${label} · 失败`;
  return `${label} · ${entry.resultCount ?? 0} 项`;
}

function SteerDetail({ entry }: { entry: SteerEntry }) {
  return (
    <ToolCard
      label={
        <>
          <MessageSquare className="h-3 w-3" />
          POST /api/review/:id/steer · stage = {entry.stage}
        </>
      }
    >
      <p className="text-sm leading-snug text-text-2">「{entry.text}」</p>
      {entry.status === "error" && entry.error ? (
        <p className="mt-2 text-[10.5px] text-high">引导失败：{entry.error}</p>
      ) : null}
      {entry.status === "done" ? (
        <p className="mt-2 text-[10.5px] text-faint">
          已替换 {entry.stage === "risks" ? "风险" : "建议"} 列表 · 共 {entry.resultCount ?? 0} 项
        </p>
      ) : null}
    </ToolCard>
  );
}

// toolMeta agent 工具调用步骤右侧 meta 文本
function toolMeta(evt: ToolEvent): string {
  if (evt.status === "running") return "运行中";
  if (evt.status === "error") return "失败";
  const len = evt.result?.length ?? 0;
  if (len > 1000) return `${(len / 1000).toFixed(1)}K chars`;
  return `${len} chars`;
}

// ToolEventDetail 工具调用详情卡：参数 + 结果预览（result 长时截断）
function ToolEventDetail({ evt }: { evt: ToolEvent }) {
  const argsPreview = evt.arguments?.slice(0, 200) ?? "";
  const result = evt.result ?? "";
  const resultPreview = result.slice(0, 600);
  const truncated = result.length > 600;
  return (
    <ToolCard
      label={
        <>
          <Wrench className="h-3 w-3" />
          {evt.name}
          {evt.arguments ? (
            <span className="ml-2 font-mono text-faint">args: {argsPreview}</span>
          ) : null}
        </>
      }
    >
      {evt.status === "running" ? (
        <p className="text-xs text-muted">执行中…</p>
      ) : evt.status === "error" ? (
        <p className="text-[11px] text-high">{result}</p>
      ) : (
        <pre className="whitespace-pre-wrap text-[11px] leading-snug text-text-2">
          {resultPreview}
          {truncated ? `\n…（已截断 ${result.length - 600} 字）` : ""}
        </pre>
      )}
    </ToolCard>
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
  const isError = status === "error";
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
                : isError
                  ? "border-high-bd bg-high-bg text-high"
                  : "border-border-strong bg-surface text-ok",
          )}
        >
          {isRunning ? (
            <Spinner size="xs" className="text-accent" />
          ) : isError ? (
            <AlertTriangle className="h-3.5 w-3.5" />
          ) : (
            icon
          )}
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
          {meta && (isDone || isError) ? (
            <span
              className={cn(
                "ml-auto whitespace-nowrap font-mono text-[10.5px]",
                isError ? "text-high" : "text-faint",
              )}
            >
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

interface BudgetView {
  items: BudgetItem[];
  total: number;
  dropped: string[];
  estimated: boolean; // true = 后端真值缺失时的粗估
}

const BUDGET_LABELS = {
  L1: "diff hunk + PR meta",
  L2: "变更文件 / 受影响函数",
  L3: "项目约定（README/CLAUDE.md）",
} as const;

// fromBudgetReport 后端真值（budget_report SSE 帧 / cached detail）→ 渲染形状
function fromBudgetReport(b: BudgetReport): BudgetView {
  const items: BudgetItem[] = [
    { layer: "L1", label: BUDGET_LABELS.L1, tok: b.used_l1, className: "bg-info" },
    { layer: "L2", label: BUDGET_LABELS.L2, tok: b.used_l2, className: "bg-ok" },
    { layer: "L3", label: BUDGET_LABELS.L3, tok: b.used_l3, className: "bg-med" },
  ];
  return {
    items,
    total: Math.max(1, b.used_l1 + b.used_l2 + b.used_l3),
    dropped: b.dropped ?? [],
    estimated: false,
  };
}

// estimateBudget 粗估：仅在后端真值未到（流式首字节前 / 旧缓存）时兜底
// L2 = patch 字节数 / 3（与后端 estimateTokens 同算法）；L1/L3 经验值
function estimateBudget(_pr: PrMeta, files: File[]): BudgetView {
  let patchChars = 0;
  for (const f of files) patchChars += f.patch?.length ?? 0;
  const l2 = Math.max(200, Math.round(patchChars / 3));
  const l1 = 400 + files.length * 30;
  const l3 = Math.min(1600, Math.round((l1 + l2) / 12));
  return {
    items: [
      { layer: "L1", label: BUDGET_LABELS.L1, tok: l1, className: "bg-info" },
      { layer: "L2", label: BUDGET_LABELS.L2, tok: l2, className: "bg-ok" },
      { layer: "L3", label: BUDGET_LABELS.L3, tok: l3, className: "bg-med" },
    ],
    total: l1 + l2 + l3,
    dropped: [],
    estimated: true,
  };
}

function ContextDetail({ budget }: { budget: BudgetView }) {
  return (
    <ToolCard
      label={
        <>
          token 预算 · L1:L2:L3 ≈ 4:5:1 · 合计 {(budget.total / 1000).toFixed(1)}K
          {budget.estimated ? <span className="text-faint"> · 粗估</span> : null}
        </>
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
      {budget.dropped.length > 0 ? (
        <div className="mt-2 border-t border-dashed border-border pt-2 text-[10.5px] text-muted">
          <span className="font-medium text-high">超预算丢弃</span>
          <span className="ml-1.5 font-mono text-faint">
            {budget.dropped.slice(0, 3).join(" · ")}
            {budget.dropped.length > 3 ? ` …+${budget.dropped.length - 3}` : ""}
          </span>
        </div>
      ) : null}
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

// SteerComposer 底部「引导」输入条；纯 UI 组件——发送 / 状态机由父级 AgentSessionView 持有。
// disabled = streaming 模式（reviewId 未知）；inFlight = 父级正在跑一次引导
function SteerComposer({
  pr,
  disabled,
  inFlight,
  onSend,
}: {
  pr: PrMeta;
  disabled: boolean;
  inFlight: boolean;
  onSend: (text: string, stage: "risks" | "suggestions", mode: SteerMode) => void;
}) {
  const [text, setText] = useState("");
  const [stage, setStage] = useState<"risks" | "suggestions">("risks");
  const [mode, setMode] = useState<SteerMode>("stage");
  const githubURL = `https://github.com/${pr.owner}/${pr.repo}/pull/${pr.pr}`;
  const enabled = !disabled && !inFlight;

  function send() {
    const v = text.trim();
    if (!v || !enabled) return;
    onSend(v, stage, mode);
    setText("");
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
          {/* 模式切换：stage（重跑 risks/suggestions）/ agent（跑 ReAct loop 调工具） */}
          <div className="flex gap-[3px] rounded-md border border-border bg-surface-2 p-[2px]">
            {(["stage", "agent"] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => setMode(m)}
                aria-pressed={mode === m}
                className={cn(
                  "rounded-sm px-2 py-[3px] font-mono text-[10.5px] transition-colors",
                  mode === m ? "bg-surface text-text" : "text-muted hover:text-text",
                )}
              >
                {m === "stage" ? "重跑 stage" : "agent 深挖"}
              </button>
            ))}
          </div>
          {/* stage 选择：仅 mode=stage 时显示；agent 模式跳过（不重跑 stage） */}
          {mode === "stage" ? (
            <div className="flex gap-[3px] rounded-md border border-border bg-surface-2 p-[2px]">
              {(["risks", "suggestions"] as const).map((s) => (
                <button
                  key={s}
                  type="button"
                  onClick={() => setStage(s)}
                  aria-pressed={stage === s}
                  className={cn(
                    "rounded-sm px-2 py-[3px] font-mono text-[10.5px] transition-colors",
                    stage === s ? "bg-surface text-text" : "text-muted hover:text-text",
                  )}
                >
                  {s === "risks" ? "重评风险" : "重出建议"}
                </button>
              ))}
            </div>
          ) : null}
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
              disabled
                ? "流式评审完成后可在此引导 agent"
                : mode === "agent"
                  ? "用 agent 深挖（例：去看看 main.go 里的并发是不是 race-free / grep 一下所有 TODO）…"
                  : `引导 agent ${stage === "risks" ? "重评风险" : "重出建议"}（例：重点看并发安全 / 忽略 style 类问题）…`
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
      </div>
    </div>
  );
}
