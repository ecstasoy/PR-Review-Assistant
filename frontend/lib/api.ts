import type { ReviewDetail, ReviewSummary } from "./types";

// postReview 已废弃 —— 后端 /api/review 改为 SSE，请改用 lib/sse.ts 的 streamReview

// listReviews 拉历史评审列表
export async function listReviews(limit?: number): Promise<ReviewSummary[]> {
  const q = typeof limit === "number" ? `?limit=${limit}` : "";
  const res = await fetch(`/api/reviews${q}`);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as { error?: string }).error ?? `HTTP ${res.status}`);
  }
  return res.json();
}

// getReview 按 id 拉详情
export async function getReview(id: string): Promise<ReviewDetail> {
  const res = await fetch(`/api/reviews/${encodeURIComponent(id)}`);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as { error?: string }).error ?? `HTTP ${res.status}`);
  }
  return res.json();
}
