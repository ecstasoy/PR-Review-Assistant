package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/review"
)

// PostReview 接收 { url }，先用 JSON 处理预检错误；
// 成功后切到 text/event-stream，按帧推送各 stage 事件。
func PostReview(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			URL string `json:"url"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "invalid request body"})
			return
		}
		url := strings.TrimSpace(body.URL)
		if url == "" {
			c.JSON(400, gin.H{"error": "url is required"})
			return
		}

		ctx := c.Request.Context()

		pr, err := d.Fetcher.Fetch(ctx, url)
		if err != nil {
			if errors.Is(err, gh.ErrInvalidPRURL) {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			slog.Error("fetch PR", "err", err, "url", url)
			c.JSON(502, gin.H{"error": "fetch upstream failed", "detail": err.Error()})
			return
		}

		// SSE 头：必须在首次 Write 之前设
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no") // 关掉 nginx / 反代缓冲

		// 首帧：PR meta —— 让前端立刻拿到 head_sha / title 显示在头部
		writeSSE(c.Writer, "pr", map[string]any{
			"id":       pr.HeadSHA,
			"owner":    pr.Owner,
			"repo":     pr.Repo,
			"pr":       pr.Number,
			"url":      url,
			"head_sha": pr.HeadSHA,
			"title":    pr.Title,
		})
		c.Writer.Flush()

		builder := d.Builder
		if builder == nil {
			builder = prctx.NewLayeredBuilder()
		}
		pCtx, err := builder.Build(pr)
		if err != nil {
			slog.Error("build prompt context", "err", err)
			writeSSE(c.Writer, "error", map[string]string{"stage": "context", "message": err.Error()})
			writeSSE(c.Writer, "done", map[string]any{})
			return
		}
		if len(pCtx.BudgetReport.Dropped) > 0 {
			slog.Warn("prctx dropped large files", "files", pCtx.BudgetReport.Dropped, "limit", pCtx.BudgetReport.TokenLimit)
		}
		merged := mergeStages(ctx, pCtx, d.Provider)

		c.Stream(func(w io.Writer) bool {
			select {
			case <-ctx.Done():
				return false
			case ev, ok := <-merged:
				if !ok {
					writeSSERaw(w, "done", json.RawMessage(`{}`))
					return false
				}
				writeSSERaw(w, ev.Type, ev.Data)
				return true
			}
		})
	}
}

// mergeStages 并发跑 summary + risks + suggestions，把各自的事件归并到一个 channel。
// 任一 stage 失败会发一帧 error event 而非中止整条流。
func mergeStages(ctx context.Context, c prctx.Context, p llm.Provider) <-chan review.Event {
	merged := make(chan review.Event, 16)
	var wg sync.WaitGroup

	stages := []review.Stage{
		review.SummaryStage{},
		review.RisksStage{},
		review.SuggestionsStage{},
	}
	wg.Add(len(stages))
	for _, s := range stages {
		go forwardStage(ctx, c, p, s, merged, &wg)
	}

	go func() {
		wg.Wait()
		close(merged)
	}()
	return merged
}

// forwardStage 跑一个 stage，把它的事件转发到 merged；ctx 取消时安全退出。
// stage 同步失败时发一帧 error 让前端能感知，而非默默丢失。
func forwardStage(ctx context.Context, c prctx.Context, p llm.Provider, s review.Stage, merged chan<- review.Event, wg *sync.WaitGroup) {
	defer wg.Done()
	events, err := s.Run(ctx, c, p)
	if err != nil {
		payload, _ := json.Marshal(map[string]string{"stage": s.Name(), "message": err.Error()})
		select {
		case <-ctx.Done():
		case merged <- review.Event{Type: "error", Data: payload}:
		}
		return
	}
	for ev := range events {
		if ev.Type == "done" {
			continue // per-stage done is suppressed; PostReview emits a single terminal done
		}
		select {
		case <-ctx.Done():
			return
		case merged <- ev:
		}
	}
}

// writeSSE 在 c.Stream 外部写一帧（首帧 pr meta 用）；调用方负责 Flush。
func writeSSE(w http.ResponseWriter, eventType string, data any) {
	raw, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, raw)
}

// writeSSERaw c.Stream 内部用；payload 已是 json.RawMessage，避免双次 Marshal。
// c.Stream 在 step 返回后自动 Flush。
// Invariant: data must be single-line JSON (no literal newlines); do not pretty-print,
// as embedded newlines would break SSE framing (each data: line must be a complete field).
func writeSSERaw(w io.Writer, eventType string, data json.RawMessage) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
}

