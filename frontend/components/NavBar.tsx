"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { History } from "lucide-react";

import { cn } from "@/lib/utils";
import { BrandMark } from "./BrandMark";
import { ThemeToggle } from "./theme-toggle";

// NavBar 顶栏：左侧 LGTM BrandMark（图标 + lgtm 字标 + 闪烁光标）→ ghost button 导航
// → mock-provider pill → 主题切换。
export function NavBar() {
  const pathname = usePathname();
  const isReview = pathname === "/" || pathname.startsWith("/review");
  const isHistory = pathname.startsWith("/history");

  return (
    <header className="flex h-[52px] flex-shrink-0 items-center gap-3 border-b border-border bg-surface px-4">
      <Link href="/" className="flex items-center" aria-label="LGTM 首页">
        <BrandMark size={24} animate />
      </Link>

      <nav className="ml-2.5 flex items-center gap-0.5">
        <NavLink href="/" active={isReview}>
          评审
        </NavLink>
        <NavLink href="/history" active={isHistory} icon={<History className="h-[13px] w-[13px]" />}>
          历史
        </NavLink>
      </nav>

      <span className="ml-auto rounded-full border border-border px-2.5 py-[3px] font-mono text-[10.5px] text-faint">
        mock provider
      </span>
      <ThemeToggle />
    </header>
  );
}

interface NavLinkProps {
  href: string;
  active: boolean;
  icon?: React.ReactNode;
  children: React.ReactNode;
}

// NavLink ghost-variant Link，active 时给 surface-hover 背景；对应 design 的 Btn active 状态
function NavLink({ href, active, icon, children }: NavLinkProps) {
  return (
    <Link
      href={href}
      className={cn(
        "inline-flex h-7 items-center gap-1.5 rounded-md px-2.5 text-xs font-medium transition-colors",
        active ? "bg-surface-hover text-text" : "text-text-2 hover:bg-surface-hover hover:text-text",
      )}
    >
      {icon}
      {children}
    </Link>
  );
}
