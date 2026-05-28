import Link from "next/link";

export function NavBar() {
  return (
    <nav className="border-b border-zinc-200 bg-white/80 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/80">
      <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-3">
        <Link href="/" className="font-semibold tracking-tight">
          PR Review Assistant
        </Link>
        <div className="flex items-center gap-4 text-sm">
          <Link href="/history" className="hover:underline">
            历史
          </Link>
          <span
            title="Coming in v2"
            className="cursor-not-allowed text-zinc-400"
          >
            Sign in
          </span>
        </div>
      </div>
    </nav>
  );
}
