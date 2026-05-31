import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";

import "./globals.css";
import { ThemeScript } from "@/components/theme-script";
import { ToastContainer } from "@/components/ui/Toast";

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
  title: "LGTM — AI 辅助代码评审",
  description: "粘贴任意 GitHub PR 链接，30 秒拿到结构化评审：变更总结 / 风险识别 / 行内建议。",
  icons: {
    icon: [
      { url: "/brand/svg/favicon.svg", type: "image/svg+xml" },
      { url: "/brand/png/favicon-32.png", sizes: "32x32", type: "image/png" },
      { url: "/brand/png/favicon-16.png", sizes: "16x16", type: "image/png" },
    ],
    apple: [{ url: "/brand/png/apple-touch-icon-180.png", sizes: "180x180" }],
  },
  manifest: "/manifest.webmanifest",
  openGraph: {
    title: "LGTM — AI 辅助代码评审",
    description: "粘贴任意 GitHub PR 链接，30 秒拿到结构化评审。",
    images: ["/brand/png/og-social.png"],
    type: "website",
  },
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
      <body className={`${geistSans.variable} ${geistMono.variable}`}>
        {children}
        {/* 全局 webhook 自动评通知 toast；任何页都能看到右下角弹窗 */}
        <ToastContainer />
      </body>
    </html>
  );
}
