"use client";

import { createContext, useCallback, useContext, useMemo, type ReactNode } from "react";

import type { Suggestion } from "@/lib/types";
import type { PermsResponse } from "@/lib/perms";

// AdoptResult 后端 /comment 或 /commit 端点成功返回字段
export interface AdoptResult {
  ok: boolean;
  comment_id?: number;
  commit_sha?: string;
  html_url?: string;
}

// AdoptContextValue 给 InlineSuggestion 用的 props 集合
// reviewId 缺失（streaming 模式）→ 所有按钮 disable
interface AdoptContextValue {
  reviewId?: string;
  perms: PermsResponse | null;
  permsLoading: boolean;
  // suggestions 完整列表；InlineSuggestion 用 findIndex 找自己 idx
  // 必须用对象身份比较，所以 caller 必须传同一个引用（不要每次 map 出新数组）
  suggestions: Suggestion[];
  // postComment 找到 idx 后调 /api/review/:id/comment/:idx
  postComment: (s: Suggestion) => Promise<AdoptResult>;
  // postCommit 占位；G6c PR 实装
  postCommit: (s: Suggestion) => Promise<AdoptResult>;
}

const AdoptContext = createContext<AdoptContextValue | null>(null);

interface ProviderProps {
  reviewId?: string;
  perms: PermsResponse | null;
  permsLoading: boolean;
  suggestions: Suggestion[];
  children: ReactNode;
}

export function AdoptProvider({ reviewId, perms, permsLoading, suggestions, children }: ProviderProps) {
  const postComment = useCallback(
    async (s: Suggestion): Promise<AdoptResult> => {
      if (!reviewId) throw new Error("评审还在流式生成中，请等结束");
      const idx = suggestions.indexOf(s);
      if (idx < 0) throw new Error("找不到建议在列表中的位置");
      const res = await fetch(`/api/review/${encodeURIComponent(reviewId)}/comment/${idx}`, {
        method: "POST",
        credentials: "include",
      });
      const data = (await res.json()) as { error?: string } & AdoptResult;
      if (!res.ok || !data.ok) {
        throw new Error(data.error || `HTTP ${res.status}`);
      }
      return data;
    },
    [reviewId, suggestions],
  );

  // G6c 之前 stub：直接 reject 让 UI 显示"功能开发中"
  const postCommit = useCallback(async (): Promise<AdoptResult> => {
    throw new Error("一键提交将在 G6c 上线，先用「💬 评论」+ GitHub UI 一键 Apply");
  }, []);

  const value = useMemo(
    () => ({ reviewId, perms, permsLoading, suggestions, postComment, postCommit }),
    [reviewId, perms, permsLoading, suggestions, postComment, postCommit],
  );

  return <AdoptContext.Provider value={value}>{children}</AdoptContext.Provider>;
}

// useAdopt 子组件用；Provider 外调返 null
export function useAdopt(): AdoptContextValue | null {
  return useContext(AdoptContext);
}
