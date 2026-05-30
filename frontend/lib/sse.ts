import type { File, PrMeta, Risk, Suggestion } from "./types";

// 重导出方便老消费者继续 import 自此处；新代码请直接从 ./types 取
export type { PrMeta } from "./types";

export interface StreamCallbacks {
  onPr?: (pr: PrMeta) => void;
  onFiles?: (files: File[]) => void;
  onSummaryDelta?: (delta: string) => void;
  onRisksDone?: (risks: Risk[]) => void;
  onSuggestionsDone?: (suggestions: Suggestion[]) => void;
  onInfo?: (message: string) => void;
  onStageError?: (stage: string, message: string) => void;
  onStageDone?: (stage: string) => void;
  onDone?: () => void;
}

// streamReview POST /api/review，按 SSE 帧分发到对应回调。
// 4xx/5xx 同步错误直接 throw；流中各 stage 错误走 onStageError。
export async function streamReview(
  url: string,
  cb: StreamCallbacks,
  signal?: AbortSignal,
): Promise<void> {
  const res = await fetch("/api/review", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
    signal,
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    const msg = (data as { error?: string }).error ?? `HTTP ${res.status}`;
    throw new Error(msg);
  }
  if (!res.body) {
    throw new Error("响应无 body");
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buf += decoder.decode(value, { stream: true });

      // 按 \n\n 切帧；最后一段可能是不完整的，留在 buf
      const parts = buf.split("\n\n");
      buf = parts.pop() ?? "";

      for (const frame of parts) {
        const parsed = parseFrame(frame);
        if (parsed) dispatch(parsed, cb);
      }
    }
  } finally {
    reader.releaseLock();
  }
}

interface ParsedFrame {
  type: string;
  data: string;
}

function parseFrame(raw: string): ParsedFrame | null {
  let type = "";
  let data = "";
  for (const line of raw.split("\n")) {
    if (line.startsWith("event: ")) {
      type = line.slice(7).trim();
    } else if (line.startsWith("data: ")) {
      data += line.slice(6);
    }
  }
  return type ? { type, data } : null;
}

function dispatch(ev: ParsedFrame, cb: StreamCallbacks): void {
  let parsed: unknown;
  try {
    parsed = JSON.parse(ev.data);
  } catch {
    return; // 非法 JSON 跳过
  }
  switch (ev.type) {
    case "pr":
      cb.onPr?.(parsed as PrMeta);
      break;
    case "files":
      cb.onFiles?.(parsed as File[]);
      break;
    case "summary_delta": {
      const p = parsed as { delta?: string };
      if (p.delta) cb.onSummaryDelta?.(p.delta);
      break;
    }
    case "risks_done":
      cb.onRisksDone?.(parsed as Risk[]);
      break;
    case "suggestions_done":
      cb.onSuggestionsDone?.(parsed as Suggestion[]);
      break;
    case "info": {
      const p = parsed as { message?: string };
      if (p.message) cb.onInfo?.(p.message);
      break;
    }
    case "error": {
      const p = parsed as { stage?: string; message?: string };
      cb.onStageError?.(p.stage ?? "?", p.message ?? "");
      break;
    }
    case "done": {
      const p = parsed as { stage?: string };
      if (p.stage) {
        cb.onStageDone?.(p.stage);
      } else {
        cb.onDone?.();
      }
      break;
    }
  }
}
