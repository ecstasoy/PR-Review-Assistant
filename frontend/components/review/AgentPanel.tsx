"use client";

import { useEffect, useRef, useState } from "react";
import { PanelRight, Send, Sparkle } from "lucide-react";

import { cn } from "@/lib/utils";
import { Avatar } from "@/components/ui/avatar";

interface Props {
  onClose: () => void;
}

interface Msg {
  role: "user" | "assistant";
  text: string;
  cites?: string[];
}

// AgentPanel 右侧 360px 抽屉 dock：追问这个 PR。
// v1 UI 完整 + beta 标，回答是占位 canned response（后端 agent 接口在 v2 实现）。
// 严格对齐 design 原型 AgentPanel.jsx 的布局。
export function AgentPanel({ onClose }: Props) {
  const [msgs, setMsgs] = useState<Msg[]>([
    {
      role: "assistant",
      text:
        "我是 PR Review Agent。基于本次评审的 diff、风险与项目约定，可以继续追问任何代码细节、设计权衡或建议落地方式。",
    },
  ]);
  const [input, setInput] = useState("");
  const [thinking, setThinking] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
  }, [msgs, thinking]);

  function send(text?: string) {
    const q = (text ?? input).trim();
    if (!q) return;
    setMsgs((m) => [...m, { role: "user", text: q }]);
    setInput("");
    setThinking(true);
    // 占位 stub —— v2 真接后端 agent 接口
    window.setTimeout(() => {
      setThinking(false);
      setMsgs((m) => [
        ...m,
        {
          role: "assistant",
          text:
            "（演示）这条回答是占位的：v1 仅实现 UI 形态，v2 会接通后端 agent 接口做真多轮对话，上下文已包含本 PR 的 diff、风险与项目约定。",
          cites: ["shard.go:45", "eviction.go:7"],
        },
      ]);
    }, 1200);
  }

  const chips = ["这个锁改动安全吗？", "采样命中率会下降吗？", "帮我写一条 review 评论"];

  return (
    <aside className="flex h-full w-[360px] shrink-0 flex-col border-l border-border bg-surface">
      <header className="flex items-center gap-2 border-b border-border px-3 py-2.5">
        <span className="inline-flex h-[22px] w-[22px] items-center justify-center rounded-md bg-accent text-accent-fg">
          <Sparkle className="h-[13px] w-[13px]" fill="currentColor" />
        </span>
        <span className="text-sm font-semibold">追问这个 PR</span>
        <span className="rounded-full border border-border bg-surface-2 px-1.5 py-px font-mono text-[10px] text-faint">
          beta
        </span>
        <button
          type="button"
          onClick={onClose}
          className="ml-auto inline-flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-surface-hover hover:text-text"
          aria-label="关闭"
        >
          <PanelRight className="h-4 w-4" />
        </button>
      </header>

      <div ref={scrollRef} className="flex-1 overflow-y-auto px-3.5 py-1.5">
        {msgs.map((m, i) => (
          <AgentMessage key={i} msg={m} />
        ))}
        {thinking ? (
          <div className="flex items-center gap-2 py-2.5">
            <span className="inline-flex h-6 w-6 items-center justify-center rounded-md bg-accent text-accent-fg">
              <Sparkle className="h-3.5 w-3.5" fill="currentColor" />
            </span>
            <span className="flex items-center gap-1">
              {[0, 1, 2].map((i) => (
                <span
                  key={i}
                  className="inline-block h-1.5 w-1.5 rounded-full bg-muted"
                  style={{ animation: `pulse-dot 1s ${i * 0.18}s infinite` }}
                />
              ))}
            </span>
          </div>
        ) : null}
      </div>

      <div className="flex flex-wrap gap-1.5 px-3 pt-2 pb-1.5">
        {chips.map((c) => (
          <button
            key={c}
            type="button"
            onClick={() => send(c)}
            className="whitespace-nowrap rounded-full border border-border bg-surface-2 px-2.5 py-1 text-[11px] text-text-2 hover:bg-surface-hover hover:text-text"
          >
            {c}
          </button>
        ))}
      </div>
      <div className="px-3 pb-3 pt-1">
        <div className="flex items-end gap-1.5 rounded-lg border border-border-strong bg-surface p-1.5">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                send();
              }
            }}
            placeholder="针对这个 PR 提问…"
            rows={1}
            className="max-h-[120px] min-w-0 flex-1 resize-none border-none bg-transparent px-1.5 py-1.5 text-sm leading-snug text-text outline-none placeholder:text-faint"
          />
          <button
            type="button"
            onClick={() => send()}
            disabled={!input.trim()}
            className="inline-flex h-7 items-center rounded-md bg-accent px-2 text-accent-fg hover:opacity-90 disabled:opacity-50"
          >
            <Send className="h-3.5 w-3.5" />
          </button>
        </div>
        <div className="mt-1.5 text-center text-[10px] text-faint">
          上下文已包含本 PR 的 diff、风险与项目约定文件
        </div>
      </div>
    </aside>
  );
}

function AgentMessage({ msg }: { msg: Msg }) {
  const isUser = msg.role === "user";
  return (
    <div
      className={cn(
        "animate-fade-up flex gap-2 py-2.5",
        isUser ? "flex-row-reverse" : "flex-row",
      )}
    >
      <span className="shrink-0">
        {isUser ? (
          <Avatar name="you" size="md" />
        ) : (
          <span className="inline-flex h-6 w-6 items-center justify-center rounded-md bg-accent text-accent-fg">
            <Sparkle className="h-3.5 w-3.5" fill="currentColor" />
          </span>
        )}
      </span>
      <div className="max-w-[82%]">
        <div
          className={cn(
            "whitespace-pre-wrap rounded-lg px-3 py-2 text-xs leading-[1.6]",
            isUser
              ? "bg-accent text-accent-fg"
              : "border border-border bg-surface-2 text-text",
          )}
        >
          {msg.text}
        </div>
        {msg.cites ? (
          <div className="mt-1.5 flex flex-wrap gap-1">
            {msg.cites.map((c) => (
              <code
                key={c}
                className="rounded border border-border bg-surface-2 px-1.5 py-px font-mono text-[10px] text-info"
              >
                {c}
              </code>
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}
