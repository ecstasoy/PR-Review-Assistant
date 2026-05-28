// 前端组件与后端 API 共享的类型
// 与 backend/internal/review 和 backend/internal/store 保持同步

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
  prNumber: number;
  headSHA: string;
  summary: string;
  risks: Risk[];
  suggestions: Suggestion[];
  createdAt: string;
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
