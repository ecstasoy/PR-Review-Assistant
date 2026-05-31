"use client";

// deleteReview DELETE /api/reviews/:id
// 后端按 ownership 拒绝：未登录 401，非 owner 403
// 成功后 caller 应刷新列表
export async function deleteReview(id: string): Promise<void> {
  const res = await fetch(`/api/reviews/${encodeURIComponent(id)}`, {
    method: "DELETE",
    credentials: "include",
  });
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(data.error || `HTTP ${res.status}`);
  }
}
