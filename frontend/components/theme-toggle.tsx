"use client";

import { useEffect, useState } from "react";
import { Moon, Sun } from "lucide-react";

import { resolveTheme, THEME_STORAGE_KEY, type Theme } from "@/lib/theme";
import { cn } from "@/lib/utils";

// ThemeToggle 切换 light / dark；data-theme 写在 <html>，配合 globals.css 的 CSS 变量
// 不用 Context：状态就是 DOM 属性 + localStorage，读写直接即可
export function ThemeToggle({ className }: { className?: string }) {
  // 初始 null 避免水合不匹配（服务器无 localStorage）；mount 后 setState 拿真实值
  const [theme, setTheme] = useState<Theme | null>(null);

  useEffect(() => {
    setTheme(resolveTheme());
  }, []);

  function toggle() {
    const next: Theme = theme === "dark" ? "light" : "dark";
    setTheme(next);
    document.documentElement.setAttribute("data-theme", next);
    try {
      window.localStorage.setItem(THEME_STORAGE_KEY, next);
    } catch {
      // localStorage 不可用就只在本次会话生效
    }
  }

  // 未水合前显示 placeholder，避免闪烁；ThemeScript 已经把正确主题写到 html 了
  if (theme === null) {
    return (
      <button
        aria-label="切换主题"
        className={cn(
          "inline-flex h-8 w-8 items-center justify-center rounded-md border border-border text-muted",
          className,
        )}
      />
    );
  }

  const Icon = theme === "dark" ? Sun : Moon;
  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={theme === "dark" ? "切到亮色" : "切到暗色"}
      title={theme === "dark" ? "切到亮色" : "切到暗色"}
      className={cn(
        "inline-flex h-8 w-8 items-center justify-center rounded-md border border-border text-muted hover:bg-surface-hover hover:text-text transition-colors",
        className,
      )}
    >
      <Icon className="h-4 w-4" />
    </button>
  );
}
