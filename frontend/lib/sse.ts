import type { SseEvent } from "./types";

// PR #12 用原生 EventSource 订阅 + 解析 SSE 事件
export function subscribeReview(
  id: string,
  onEvent: (e: SseEvent) => void,
): () => void {
  void id;
  void onEvent;
  throw new Error("subscribeReview: not implemented yet (PR #12)");
}
