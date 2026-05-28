import type { ReviewResult } from "./types";

// PR #5 接 POST /api/review
export async function postReview(url: string): Promise<ReviewResult> {
  throw new Error("postReview: not implemented yet (PR #5)");
}

// PR #14 接 GET /api/reviews
export async function listReviews(): Promise<ReviewResult[]> {
  throw new Error("listReviews: not implemented yet (PR #14)");
}

// PR #15 接 GET /api/reviews/:id
export async function getReview(id: string): Promise<ReviewResult> {
  throw new Error(`getReview(${id}): not implemented yet (PR #15)`);
}
