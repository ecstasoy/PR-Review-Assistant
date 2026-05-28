import type { Risk, Suggestion, SseEvent } from "./types";

export interface ReviewState {
  summary: string;
  risks: Risk[];
  suggestions: Suggestion[];
  status: "idle" | "streaming" | "done" | "error";
  error?: string;
}

export const initialReviewState: ReviewState = {
  summary: "",
  risks: [],
  suggestions: [],
  status: "idle",
};

// PR #12 按事件类型分支细化
export function reviewReducer(
  state: ReviewState,
  event: SseEvent,
): ReviewState {
  switch (event.type) {
    case "summary_delta":
      return { ...state, status: "streaming" };
    case "risks_done":
      return { ...state, status: "streaming" };
    case "suggestions_done":
      return { ...state, status: "streaming" };
    case "done":
      return { ...state, status: "done" };
    case "error":
      return {
        ...state,
        status: "error",
        error: typeof event.data === "string" ? event.data : "unknown error",
      };
    default:
      return state;
  }
}
