// 主题切换的常量 + 序列化逻辑；inline script 和 client toggle 共用
// 选 data-theme 而不是 class，因为 CSS 变量用 [data-theme="dark"] 选择器更直接

export type Theme = "light" | "dark";
export const THEME_STORAGE_KEY = "pr-review-theme";

// resolveTheme 按 localStorage → prefers-color-scheme → fallback 顺序确定主题。
// 仅在浏览器调用（依赖 window / matchMedia）
export function resolveTheme(): Theme {
  if (typeof window === "undefined") return "light";
  try {
    const saved = window.localStorage.getItem(THEME_STORAGE_KEY);
    if (saved === "light" || saved === "dark") return saved;
  } catch {
    // localStorage 不可用（隐私模式 / 沙盒）→ 落到系统偏好
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}
