import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";

import "./globals.css";
import { NavBar } from "@/components/NavBar";

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

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    // 默认 light；后续主题切换 commit 加 no-flash inline script 在 SSR 时按 localStorage 设置
    <html lang="zh-CN" data-theme="light" data-density="comfortable">
      <body className={`${geistSans.variable} ${geistMono.variable}`}>
        <NavBar />
        <main className="mx-auto max-w-5xl px-6 py-8">{children}</main>
      </body>
    </html>
  );
}
