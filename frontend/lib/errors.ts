// friendlyError 把后端 / 网络 / 超时的原始错误信息翻成对用户可操作的中文。
// 后端对 403/404 已返回中文（见 backend/internal/api/review.go），这里只补齐英文兜底、
// 网络层失败和 SSE 空闲超时；其余原样透出（含后端已中文化的提示）。
export function friendlyError(raw: string): string {
  const m = raw.trim();
  if (!m) return "评审失败，请重试。";

  // SSE 空闲超时（lib/sse.ts SSETimeoutError）
  if (m === "sse idle timeout" || m.includes("idle timeout")) {
    return "评审响应超时——可能是 PR 过大或服务繁忙，请重试。";
  }
  // fetch 网络层失败（离线 / DNS / CORS）：各浏览器文案不一
  if (m === "Failed to fetch" || m.includes("NetworkError") || m === "Load failed") {
    return "网络连接失败，请检查网络后重试。";
  }
  // 无效 PR 链接（后端 gh.ErrInvalidPRURL / 请求体校验）
  if (
    m.includes("invalid GitHub PR URL") ||
    m === "invalid request body" ||
    m === "url is required"
  ) {
    return "PR 链接无效，请填形如 https://github.com/owner/repo/pull/123 的地址。";
  }
  // 上游拉取失败（502）
  if (m.includes("fetch upstream failed")) {
    return "拉取 PR 失败，请稍后重试。";
  }
  // 其余（含后端已中文化的 403 / 404）原样透出
  return m;
}
