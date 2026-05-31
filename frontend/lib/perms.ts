"use client";

import { useEffect, useState } from "react";

// PermsResponse /api/perms 返回；同后端 api.PermsResponse
export interface PermsResponse {
  authenticated: boolean;
  permission?: string;
  can_comment: boolean;
  can_commit: boolean;
  reason?: string;
}

// usePerms 拉指定 repo 的执行权限；owner/repo 为空时跳过 fetch
// 不长期缓存：每次进入 review 页拉一次足够；登录态变更也由父组件 force remount
export function usePerms(owner?: string, repo?: string): {
  perms: PermsResponse | null;
  loading: boolean;
} {
  const [perms, setPerms] = useState<PermsResponse | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!owner || !repo) {
      setPerms(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    const q = `?owner=${encodeURIComponent(owner)}&repo=${encodeURIComponent(repo)}`;
    fetch("/api/perms" + q, { credentials: "include" })
      .then((r) => r.json() as Promise<PermsResponse>)
      .then((data) => {
        if (!cancelled) setPerms(data);
      })
      .catch(() => {
        if (!cancelled) {
          setPerms({
            authenticated: false,
            can_comment: false,
            can_commit: false,
            reason: "权限查询失败（网络错）",
          });
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [owner, repo]);

  return { perms, loading };
}
