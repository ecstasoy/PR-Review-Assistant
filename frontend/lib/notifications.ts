"use client";

import { useEffect, useRef, useState } from "react";

// Notification 后端 /api/notifications 返回的字段；同 api.Notification 形状
export interface Notification {
  id: string;
  review_id: string;
  owner: string;
  repo: string;
  pr: number;
  title?: string;
  source: string; // "webhook"
  created_at: string;
}

// useNotifications 轮询拉新 webhook 通知
// since 维护：每次拉到后记最新 id，下次只拉更新的，避免重复弹 toast
//
// 仅在用户登录后开始轮询（依赖 cookie；未登录返空）
// 后续可改 SSE/WebSocket 推送；当前 15s 轮询足够 demo
export function useNotifications(intervalMs = 15000): {
  newOnes: Notification[];
  consume: () => void;
} {
  const [newOnes, setNewOnes] = useState<Notification[]>([]);
  const sinceRef = useRef<string | null>(null);
  // 用 ref 防 effect 闭包陷阱
  const intervalRef = useRef<number | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function tick() {
      try {
        const q = sinceRef.current
          ? `?since=${encodeURIComponent(sinceRef.current)}`
          : "";
        const res = await fetch("/api/notifications" + q, {
          credentials: "include",
        });
        if (!res.ok) return;
        const list = (await res.json()) as Notification[];
        if (cancelled || list.length === 0) return;
        // 列表是后端按时间倒序；最新的在 [0]
        sinceRef.current = list[0].id;
        setNewOnes((prev) => [...list, ...prev]);
      } catch {
        // 网络挂了 / 未登录返 200 空；安静失败
      }
    }

    // 立刻拉一次拿到 baseline since（避免上线后被旧通知淹没）
    void tick();
    intervalRef.current = window.setInterval(tick, intervalMs);

    return () => {
      cancelled = true;
      if (intervalRef.current !== null) {
        window.clearInterval(intervalRef.current);
      }
    };
  }, [intervalMs]);

  function consume() {
    setNewOnes([]);
  }

  return { newOnes, consume };
}
