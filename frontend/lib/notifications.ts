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
//
// 首次 tick：静默 baseline —— 只把当前最新条目 ID 记到 sinceRef，**不**推到 newOnes 弹 toast
// 后续 tick：只拉 since 之后的，真正"新到"的弹 toast
//
// 目的：刷新页面 / 关 tab 重开不再把过去 7 天积压的通知一次性灌成 toast 雪崩
// 代价：用户离开页面期间到的通知，再回来不显示（除非装 localStorage 持久 sinceRef，本期未做）
//
// 仅在用户登录后开始轮询（依赖 cookie；未登录后端返空）
// 后续可改 SSE/WebSocket 推送；当前 15s 轮询足够 demo
export function useNotifications(intervalMs = 15000): {
  newOnes: Notification[];
  consume: () => void;
} {
  const [newOnes, setNewOnes] = useState<Notification[]>([]);
  const sinceRef = useRef<string | null>(null);
  const initialBaselineRef = useRef<boolean>(true); // 首次 tick 标记，仅记 baseline 不弹
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
        if (cancelled) return;

        // 首次 tick：只记 baseline，不弹（防刷新雪崩）
        if (initialBaselineRef.current) {
          initialBaselineRef.current = false;
          if (list.length > 0) {
            sinceRef.current = list[0].id;
          }
          return;
        }

        if (list.length === 0) return;
        // 列表后端按时间倒序；最新的在 [0]
        sinceRef.current = list[0].id;
        setNewOnes((prev) => [...list, ...prev]);
      } catch {
        // 网络挂 / 未登录返 200 空；安静失败
      }
    }

    // 立刻 tick 一次拿 baseline（不弹任何 toast）
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
