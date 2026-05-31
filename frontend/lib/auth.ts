"use client";

import { useEffect, useState } from "react";

// Me /api/me 返回字段；与后端 api.MeResponse 同形状
export interface Me {
  authenticated: boolean;
  login?: string;
  user_id?: number;
  avatar_url?: string;
  name?: string;
}

// useMe 拉登录态；nullable + loading 区分初始未拉 vs 未登录
// 单次 fetch；登录 / 登出会刷新页面，无需 polling
export function useMe(): { me: Me | null; loading: boolean; reload: () => void } {
  const [me, setMe] = useState<Me | null>(null);
  const [loading, setLoading] = useState(true);
  const [nonce, setNonce] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetch("/api/me", { credentials: "include" })
      .then((r) => r.json() as Promise<Me>)
      .then((data) => {
        if (!cancelled) setMe(data);
      })
      .catch(() => {
        if (!cancelled) setMe({ authenticated: false });
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [nonce]);

  return { me, loading, reload: () => setNonce((n) => n + 1) };
}

// signInURL 当前页面作为 next 参数；GitHub 登录后跳回原页
export function signInURL(): string {
  if (typeof window === "undefined") return "/api/auth/github/login";
  const next = encodeURIComponent(window.location.pathname + window.location.search);
  return `/api/auth/github/login?next=${next}`;
}

// signOut POST /api/auth/logout 然后刷新当前页
export async function signOut() {
  await fetch("/api/auth/logout", { method: "POST", credentials: "include" });
  if (typeof window !== "undefined") {
    window.location.reload();
  }
}
