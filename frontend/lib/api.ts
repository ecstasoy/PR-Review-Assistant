import type { ReviewResult } from "./types";

const BASE = "/api";

// postReview 调 POST /api/review，返回 LLM 总结结果。
// 错误响应（4xx/5xx）抛出 Error，message 取后端 { error } 字段。
export async function postReview(url: string): Promise<ReviewResult> {
  const res = await fetch(`${BASE}/review`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = (data as { error?: string }).error ?? `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return data as ReviewResult;
}

// 后续 PR 接 GET /api/reviews
export async function listReviews(): Promise<ReviewResult[]> {
  throw new Error("listReviews: not implemented yet");
}

// 后续 PR 接 GET /api/reviews/:id
export async function getReview(id: string): Promise<ReviewResult> {
  throw new Error(`getReview(${id}): not implemented yet`);
}
