"use client";

import { useState } from "react";
import { Check, Clipboard, Sparkle, X } from "lucide-react";

import type { Suggestion } from "@/lib/types";
import { cn } from "@/lib/utils";
import { CategoryBadge, type Category } from "@/components/ui/badge";

interface Props {
  suggestion: Suggestion;
  onCopy?: () => void;
}

// InlineSuggestion 行内建议气泡（DiffView 内嵌锚定到对应代码行）
// 严格对齐 design 原型 Diff.jsx 的 InlineSuggestion：
// surface-2 底 + 左 padding 50px 与代码列对齐；header AI chip + CategoryTag；
// 标题 + body + 可选 before/after patch 块（红绿带 - / + 前缀）；3 操作按钮
export function InlineSuggestion({ suggestion, onCopy }: Props) {
  const [copied, setCopied] = useState(false);
  const [applied, setApplied] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  if (dismissed) return null;

  function copy() {
    const md = formatAsMarkdown(suggestion);
    navigator.clipboard.writeText(md).then(() => {
      setCopied(true);
      onCopy?.();
      setTimeout(() => setCopied(false), 1400);
    });
  }

  return (
    <div className="animate-fade-up border-y border-border bg-surface-2 py-2.5 pl-[50px] pr-3.5">
      <div className="mb-1.5 flex items-center gap-[7px]">
        <span className="inline-flex items-center gap-1 rounded-[5px] bg-accent-soft px-[7px] py-0.5 font-mono text-xs font-semibold text-accent">
          <Sparkle className="h-3 w-3" fill="currentColor" />
          AI 建议
        </span>
        <CategoryBadge category={suggestion.type as Category} />
      </div>
      <div className="mb-1 text-sm font-semibold text-text">{suggestion.title}</div>
      <div className={cn("text-xs leading-[1.6] text-text-2", suggestion.patch ? "mb-2.5" : "mb-1")}>
        {suggestion.body}
      </div>

      {suggestion.patch ? (
        <div className="overflow-hidden rounded-md border border-border bg-surface font-mono text-[12.5px]">
          {suggestion.patch.before.split("\n").map((l, i) => (
            <div
              key={`b-${i}`}
              className="whitespace-pre-wrap bg-del-bg px-3 py-px"
            >
              <span className="select-none text-fail">- </span>
              {l}
            </div>
          ))}
          {suggestion.patch.after.split("\n").map((l, i) => (
            <div
              key={`a-${i}`}
              className="whitespace-pre-wrap bg-add-bg px-3 py-px"
            >
              <span className="select-none text-ok">+ </span>
              {l}
            </div>
          ))}
        </div>
      ) : null}

      <div className="mt-2.5 flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setApplied(true)}
          className="inline-flex h-7 items-center gap-1 rounded-md bg-accent px-2.5 text-xs font-medium text-accent-fg hover:opacity-90"
        >
          <Check className="h-3 w-3" />
          {applied ? "已采纳" : "采纳建议"}
        </button>
        <button
          type="button"
          onClick={copy}
          className="inline-flex h-7 items-center gap-1 rounded-md border border-border-strong bg-surface px-2.5 text-xs text-text-2 hover:bg-surface-hover hover:text-text"
        >
          {copied ? <Check className="h-3 w-3 text-ok" /> : <Clipboard className="h-3 w-3" />}
          {copied ? "已复制" : "复制到 GitHub"}
        </button>
        <button
          type="button"
          onClick={() => setDismissed(true)}
          className="inline-flex h-7 items-center gap-1 rounded-md px-2.5 text-xs text-muted hover:text-text"
        >
          <X className="h-3 w-3" />
          忽略
        </button>
      </div>
    </div>
  );
}

// formatAsMarkdown 把建议格式化成可直接照搬进 GitHub PR comment 的 markdown
function formatAsMarkdown(s: Suggestion): string {
  const parts: string[] = [];
  parts.push(`**${s.title}**`);
  parts.push("");
  parts.push(s.body);
  if (s.patch) {
    parts.push("");
    parts.push("```" + (s.patch.lang || ""));
    parts.push("// before");
    parts.push(s.patch.before);
    parts.push("// after");
    parts.push(s.patch.after);
    parts.push("```");
  }
  return parts.join("\n");
}
