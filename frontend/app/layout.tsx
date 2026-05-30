import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";

import "./globals.css";
import { ThemeScript } from "@/components/theme-script";

// next/font 注入 CSS 变量；globals.css 的 --font-sans / --font-mono 引用这两个
const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "PR Review Assistant",
  description: "AI 辅助代码评审 — 粘贴 GitHub PR 链接即可。",
};

// 根布局只负责 html / body / 字体 / 主题脚本 / 全局 CSS。
// 不挂 NavBar 也不限宽——/review/[id] 走 edge-to-edge 全宽 dashboard 风格，
// landing / history 在 app/(main)/layout.tsx 里加 NavBar + 居中限宽。
export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="zh-CN" data-theme="light" data-density="comfortable" suppressHydrationWarning>
      <head>
        <ThemeScript />
      </head>
      <body className={`${geistSans.variable} ${geistMono.variable}`}>{children}</body>
    </html>
  );
}
