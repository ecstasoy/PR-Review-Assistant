"use client";

import { useState } from "react";

export function ReviewForm() {
  const [url, setUrl] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // PR #4 接通：POST /api/review → 跳转到 /review/[id]
  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("not implemented yet (PR #4)");
    setSubmitting(false);
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <label className="block">
        <span className="block text-sm font-medium">GitHub PR URL</span>
        <input
          type="url"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://github.com/owner/repo/pull/123"
          className="mt-1 w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm shadow-sm outline-none focus:border-zinc-900 dark:border-zinc-700 dark:bg-zinc-900 dark:focus:border-zinc-100"
          required
        />
      </label>
      <button
        type="submit"
        disabled={submitting || !url}
        className="rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
      >
        {submitting ? "分析中…" : "开始评审"}
      </button>
      {error ? (
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      ) : null}
    </form>
  );
}
