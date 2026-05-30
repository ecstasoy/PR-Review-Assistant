import { NavBar } from "@/components/NavBar";

// (main) route group 布局：用于 landing (`/`) 和 history (`/history`)。
// 顶部全局 NavBar + 主区居中 + max-w-5xl 限宽。
// /review/[id] 不走这套布局（不在 (main) 下），自己控制 edge-to-edge dashboard 排版。
export default function MainGroupLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <>
      <NavBar />
      <main className="mx-auto max-w-5xl px-6 py-8">{children}</main>
    </>
  );
}
