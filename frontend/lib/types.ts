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

export interface Suggestion {
  file: string;
  line: number;
  type: "bug" | "style" | "perf" | "security" | "concurrency";
  title: string;
  body: string;
  patch?: Patch | null; // LLM 给不出具体代码改写时为 null / 省略
}

export interface ReviewResult {
  id: string;
  owner: string;
  repo: string;
  pr: number;
  url: string;
  head_sha: string;
  title: string;
  summary: string;
  risks?: Risk[];          // 后续 PR 填
  suggestions?: Suggestion[]; // 后续 PR 填
}

// ReviewSummary /api/reviews 列表项；不含 payload
export interface ReviewSummary {
  id: string;
  owner: string;
  repo: string;
  pr: number;
  head_sha: string;
  title?: string;
  created_at: string; // RFC3339
}

// ReviewDetail /api/reviews/:id 详情；inline 缓存 payload
export interface ReviewDetail extends ReviewSummary {
  summary: string;
  risks?: Risk[];
  suggestions?: Suggestion[];
}

export type EventType =
  | "summary_delta"
  | "risks_done"
  | "suggestions_done"
  | "error"
  | "done";

export interface SseEvent {
  type: EventType;
  data: unknown;
}
