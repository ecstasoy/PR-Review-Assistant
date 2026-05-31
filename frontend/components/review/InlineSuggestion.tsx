"use client";

import { useState } from "react";
import { Check, Clipboard, ExternalLink, GitCommit, MessageSquare, Sparkle, X } from "lucide-react";

import type { Suggestion } from "@/lib/types";
import { cn } from "@/lib/utils";
import { signInURL } from "@/lib/auth";
import { CategoryBadge, type Category } from "@/components/ui/badge";
import { useAdopt } from "./AdoptContext";

interface Props {
  suggestion: Suggestion;
  onCopy?: () => void;
}

type CommentState =
  | { kind: "idle" }
  | { kind: "posting" }
  | { kind: "done"; url: string }
  | { kind: "error"; msg: string };

// InlineSuggestion 行内建议气泡（DiffView 内嵌锚定到对应代码行）
// 严格对齐 design 原型 Diff.jsx 的 InlineSuggestion：surface-2 底 + 左 padding 50px
// 4 按钮：💬 评论 (G6b)、✅ 提交 (G6c stub)、📋 复制 markdown、✕ 忽略
// 评论/提交按钮按 useAdopt() 返的 perms 状态 disable + 悬浮 tooltip 说明原因
export function InlineSuggestion({ suggestion, onCopy }: Props) {
  const adopt = useAdopt();
  const [copied, setCopied] = useState(false);
  const [comment, setComment] = useState<CommentState>({ kind: "idle" });
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

  async function postComment() {
    if (!adopt) return;
    setComment({ kind: "posting" });
    try {
      const r = await adopt.postComment(suggestion);
      setComment({ kind: "done", url: r.html_url ?? "" });
    } catch (e) {
      setComment({ kind: "error", msg: e instanceof Error ? e.message : String(e) });
    }
  }

  // 按钮可用性判定：perms 决定（缺登录 / 无权限 / streaming 等）
  const perms = adopt?.perms ?? null;
  const reviewReady = !!adopt?.reviewId;
  const authenticated = !!perms?.authenticated;

  const commentEnabled = reviewReady && authenticated && (perms?.can_comment ?? false);
  const commitEnabled = reviewReady && authenticated && (perms?.can_commit ?? false);

  // tooltip 文案：按状态优先级返
  function disableReason(): string {
    if (!reviewReady) return "评审还在流式生成中，等结束后可用";
    if (!authenticated) return "登录后才能直接发到 GitHub";
    if (perms?.reason) return perms.reason;
    return "";
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
            <div key={`b-${i}`} className="whitespace-pre-wrap bg-del-bg px-3 py-px">
              <span className="select-none text-fail">- </span>
              {l}
            </div>
          ))}
          {suggestion.patch.after.split("\n").map((l, i) => (
            <div key={`a-${i}`} className="whitespace-pre-wrap bg-add-bg px-3 py-px">
              <span className="select-none text-ok">+ </span>
              {l}
            </div>
          ))}
        </div>
      ) : null}

      <div className="mt-2.5 flex flex-wrap items-center gap-2">
        {/* 💬 评论：G6b */}
        {commentEnabled ? (
          <button
            type="button"
            onClick={postComment}
            disabled={comment.kind === "posting"}
            className="inline-flex h-7 items-center gap-1 rounded-md bg-accent px-2.5 text-xs font-medium text-accent-fg hover:opacity-90 disabled:opacity-60"
            title="作为 PR review comment 发到 GitHub（含一键 Apply 块）"
          >
            <MessageSquare className="h-3 w-3" />
            {comment.kind === "posting" ? "发送中…" : "评论到 PR"}
          </button>
        ) : !authenticated && reviewReady ? (
          <a
            href={signInURL()}
            className="inline-flex h-7 items-center gap-1 rounded-md bg-accent px-2.5 text-xs font-medium text-accent-fg hover:opacity-90"
            title="GitHub 登录后即可直接发到 PR"
          >
            <MessageSquare className="h-3 w-3" />
            登录后评论
          </a>
        ) : (
          <button
            type="button"
            disabled
            title={disableReason()}
            className="inline-flex h-7 items-center gap-1 rounded-md bg-accent px-2.5 text-xs font-medium text-accent-fg opacity-50 cursor-not-allowed"
          >
            <MessageSquare className="h-3 w-3" />
            评论到 PR
          </button>
        )}

        {/* ✅ 提交：G6c 预留 stub */}
        <button
          type="button"
          disabled
          title={
            commitEnabled
              ? "一键提交将在 G6c 上线"
              : disableReason() || "无 push 权限"
          }
          className="inline-flex h-7 items-center gap-1 rounded-md border border-border-strong bg-surface px-2.5 text-xs text-muted opacity-60 cursor-not-allowed"
        >
          <GitCommit className="h-3 w-3" />
          直接提交
        </button>

        {/* 📋 复制：永远可用 */}
        <button
          type="button"
          onClick={copy}
          className="inline-flex h-7 items-center gap-1 rounded-md border border-border-strong bg-surface px-2.5 text-xs text-text-2 hover:bg-surface-hover hover:text-text"
        >
          {copied ? <Check className="h-3 w-3 text-ok" /> : <Clipboard className="h-3 w-3" />}
          {copied ? "已复制" : "复制 markdown"}
        </button>

        {/* ✕ 忽略：本地隐藏 */}
        <button
          type="button"
          onClick={() => setDismissed(true)}
          className="inline-flex h-7 items-center gap-1 rounded-md px-2.5 text-xs text-muted hover:text-text"
        >
          <X className="h-3 w-3" />
          忽略
        </button>
      </div>

      {/* 评论后反馈 */}
      {comment.kind === "done" ? (
        <p className="mt-2 inline-flex items-center gap-1.5 text-[11px] text-ok">
          <Check className="h-3 w-3" />
          已发到 PR
          {comment.url ? (
            <a
              href={comment.url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-0.5 underline hover:opacity-80"
            >
              查看
              <ExternalLink className="h-2.5 w-2.5" />
            </a>
          ) : null}
        </p>
      ) : null}
      {comment.kind === "error" ? (
        <p className="mt-2 text-[11px] text-high" title={comment.msg}>
          发送失败：{comment.msg}
        </p>
      ) : null}
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
