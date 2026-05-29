package api

import (
	"encoding/json"
	"net/http"
)

// cachedPayload 缓存的 review 内容。
// summary 存累加后的全文；risks / suggestions 存 stage 原 event data 字节，
// 让回放只需"原样写回"即可，避免与 review 包的具体类型耦合。
type cachedPayload struct {
	Summary     string          `json:"summary"`
	Risks       json.RawMessage `json:"risks"`
	Suggestions json.RawMessage `json:"suggestions"`
}

// replayCached 把缓存内容按 SSE 协议依次写回；调用方负责事先已发首帧 pr meta。
// 在 c.Stream 外手写，因此最后手动 Flush。
// 不发 info / cached 标记事件：前端 info 语义是"短路隐藏 sections"，发了反而藏住缓存内容；
// 用户体感"秒回"即缓存信号，UI badge 留后续 PR。
func replayCached(w http.ResponseWriter, p cachedPayload) {
	if p.Summary != "" {
		// 单帧 delta 即可拼出完整 summary（前端 reducer 是 += 累加）
		writeSSE(w, "summary_delta", map[string]string{"delta": p.Summary})
	}
	if len(p.Risks) > 0 {
		writeSSERaw(w, "risks_done", p.Risks)
	}
	if len(p.Suggestions) > 0 {
		writeSSERaw(w, "suggestions_done", p.Suggestions)
	}
	writeSSE(w, "done", map[string]any{})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
