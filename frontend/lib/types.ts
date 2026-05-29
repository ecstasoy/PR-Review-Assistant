// 前端与后端共享的类型
// 字段命名与后端 JSON 一致（snake_case），避免转换

export interface Risk {
  file: string;
  line?: number;
  severity: "high" | "medium" | "low";
  category: "bug" | "security" | "perf" | "style" | "other";
  reason: string;
}

export interface Suggestion {
  file: string;
  line: number;
  type: "bug" | "style" | "perf" | "security";
  suggestion: string;
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
