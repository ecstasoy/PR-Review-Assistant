// 前端与后端共享的类型
// 字段命名与后端 JSON 一致（snake_case），避免转换

export interface Risk {
  file: string;
  line?: number;
  severity: "high" | "medium" | "low";
  category: "bug" | "security" | "perf" | "style" | "concurrency" | "other";
  confidence: number; // 0-1，LLM 自评把握度；≥ 0.9 默认展开
  reason: string;
}

export interface Patch {
  lang: string;
  before: string;
  after: string;
}

// File PR 改动文件；raw unified diff text 在 patch 字段
export interface File {
  path: string;
  status: "added" | "modified" | "removed" | "renamed";
  patch: string;
  additions: number;
  deletions: number;
}

export interface Suggestion {
  file: string;
  line: number;
  type: "bug" | "style" | "perf" | "security" | "concurrency";
  title: string;
  body: string;
  patch?: Patch | null; // LLM 给不出具体代码改写时为 null / 省略
}

// Stats PR 体量统计
export interface Stats {
  files: number;
  additions: number;
  deletions: number;
  commits: number;
  comments: number;
}

// Check 单个 CI 检查项
export interface Check {
  name: string;
  status: "passing" | "failing" | "pending" | string; // 容忍未知
  duration_ms: number;
  note?: string; // check-run output.summary（如 coverage "82.4% (-0.3%)"）
}

// PrMeta 评审 SSE pr 事件 / detail 共享的 PR 元信息
// 字段对齐 gh.PullRequest (A1/A2/A3/author_role PR) 经 prMetaPayload + cachedPayload 透出的形状
export interface PrMeta {
  id: string;
  owner: string;
  repo: string;
  pr: number;
  url: string;
  head_sha: string;
  title: string;
  author?: string;
  author_role?: string; // OWNER / MEMBER / COLLABORATOR / CONTRIBUTOR / FIRST_TIMER / NONE
  state?: string; // open / closed / merged
  labels?: string[];
  base_ref?: string;
  head_ref?: string;
  pr_created_at?: string; // RFC3339
  stats?: Stats;
  ci?: "passing" | "failing" | "pending" | string;
  checks?: Check[];
  // source 仅 detail 端点会返；前端用来在顶栏渲染 ⚡ 自动 chip
  // streaming 期间 onPr 不带，所以 optional
  source?: "manual" | "webhook";
}

// BudgetReport 三层上下文 token 预算实际分配；后端 prctx.LayeredBuilder 输出 + SSE budget_report 帧 + detail.budget_report 同形状
export interface BudgetReport {
  token_limit: number;
  used_l1: number;
  used_l2: number;
  used_l3: number;
  used_l4?: number; // v3 RAG 才用
  dropped?: string[]; // 因预算丢弃全文的文件路径
}

// ReviewSummary /api/reviews 列表项；不含 payload
export interface ReviewSummary {
  id: string;
  owner: string;
  repo: string;
  pr: number;
  head_sha: string;
  title?: string;
  created_at: string; // RFC3339（评审记录创建时间）
  ci?: string;
  lang?: string; // PR 主语言（Go / TypeScript / Python / …）；后端按文件后缀多数派算
  source?: "manual" | "webhook"; // webhook 触发的自动评审；列表渲染 ⚡ chip
  created_by?: string; // GitHub login；空 = 匿名遗留；前端用来 gate 删除按钮
  risk_counts?: { high: number; medium: number; low: number };
}

// ReviewDetail /api/reviews/:id 详情；inline 缓存 payload + 全套 PR meta
// 注意：source 通过 ReviewSummary 继承；前端可直接 detail.source 读
export interface ReviewDetail extends ReviewSummary {
  // PR meta（A1+A2+A3+author_role+lang）
  author?: string;
  author_role?: string;
  state?: string;
  labels?: string[];
  base_ref?: string;
  head_ref?: string;
  pr_created_at?: string;
  stats?: Stats;
  checks?: Check[];

  files?: File[];
  summary: string;
  risks?: Risk[];
  suggestions?: Suggestion[];
  budget_report?: BudgetReport;
}

// 兼容旧导出（lib/api.ts ReviewResult 类型已无主动消费者）
export interface ReviewResult {
  id: string;
  owner: string;
  repo: string;
  pr: number;
  url: string;
  head_sha: string;
  title: string;
  summary: string;
  risks?: Risk[];
  suggestions?: Suggestion[];
}

// AgentToolCall agent loop 单次工具调用（与后端 SSE tool_call_start/done 帧 data 字段对应）
export interface AgentToolCall {
  id: string;
  name: string;
  arguments?: string; // start 帧含；done 帧不重复
  result?: string;    // done 帧含；含 "error: ..." 字符串
}

export type EventType =
  | "summary_delta"
  | "risks_done"
  | "suggestions_done"
  | "steered_risks_done"
  | "steered_suggestions_done"
  | "files"
  | "budget_report"
  | "tool_call_start"
  | "tool_call_done"
  | "agent_text_delta"
  | "info"
  | "error"
  | "review_id"
  | "done";

export interface SseEvent {
  type: EventType;
  data: unknown;
}
