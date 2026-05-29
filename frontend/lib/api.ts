import type { ReviewResult } from "./types";

// postReview 已废弃 —— 后端 /api/review 改为 SSE，请改用 lib/sse.ts 的 streamReview
// 后续 PR 接 GET /api/reviews
export async function listReviews(): Promise<ReviewResult[]> {
  throw new Error("listReviews: not implemented yet");
}

// 后续 PR 接 GET /api/reviews/:id
export async function getReview(id: string): Promise<ReviewResult> {
  throw new Error(`getReview(${id}): not implemented yet`);
}
