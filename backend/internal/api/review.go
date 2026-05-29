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
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/store"
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
			switch {
			case errors.Is(err, gh.ErrInvalidPRURL):
				c.JSON(400, gin.H{"error": err.Error()})
			case errors.Is(err, gh.ErrPRNotFound):
				c.JSON(404, gin.H{"error": "PR 不存在或为私有仓库（请配置 GITHUB_TOKEN）"})
			case errors.Is(err, gh.ErrAccessDenied):
				c.JSON(403, gin.H{"error": "GitHub 拒绝访问（速率限制或权限不足）"})
			default:
				slog.Error("fetch PR", "err", err, "url", url)
				c.JSON(502, gin.H{"error": "fetch upstream failed", "detail": err.Error()})
			}
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

		// 空 PR 短路：没有可评审的文件改动时，不跑 LLM，直接发 info + done
		if len(pr.Files) == 0 {
			writeSSE(c.Writer, "info", map[string]string{"message": "该 PR 无可评审的文件改动"})
			writeSSE(c.Writer, "done", map[string]any{})
			c.Writer.Flush()
			return
		}

		// 缓存命中：同 (owner, repo, pr, head_sha) 有完整结果直接回放，跳过 LLM
		if d.Store != nil {
			if rec, gerr := d.Store.Get(ctx, pr.Owner, pr.Repo, pr.Number, pr.HeadSHA); gerr != nil {
				slog.Warn("cache get failed; falling through to stages", "err", gerr)
			} else if rec != nil {
				var p cachedPayload
				if uerr := json.Unmarshal(rec.Payload, &p); uerr != nil {
					slog.Warn("cached payload unmarshal failed; falling through to stages", "err", uerr, "id", rec.ID)
				} else if p.Risks == nil || p.Suggestions == nil || !json.Valid(p.Risks) || !json.Valid(p.Suggestions) {
					slog.Warn("cached payload incomplete/invalid; falling through to stages", "id", rec.ID)
				} else {
					replayCached(c.Writer, p)
					return
				}
			}
		}

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

		// 边推流边收集供后续 cache 写入；stage 任一报错则不写缓存（避免缓存半残结果）
		var (
			summaryBuf       strings.Builder
			risksData        json.RawMessage
			suggestionsData  json.RawMessage
			stageErrObserved bool
		)
		c.Stream(func(w io.Writer) bool {
			select {
			case <-ctx.Done():
				return false
			case ev, ok := <-merged:
				if !ok {
					if d.Store != nil && !stageErrObserved && risksData != nil && suggestionsData != nil {
						persistReview(d.Store, pr, summaryBuf.String(), risksData, suggestionsData)
					}
					writeSSERaw(w, "done", json.RawMessage(`{}`))
					return false
				}
				switch ev.Type {
				case "summary_delta":
					var p struct {
						Delta string `json:"delta"`
					}
					_ = json.Unmarshal(ev.Data, &p)
					summaryBuf.WriteString(p.Delta)
				case "risks_done":
					risksData = ev.Data
				case "suggestions_done":
					suggestionsData = ev.Data
				case "error":
					stageErrObserved = true
				}
				writeSSERaw(w, ev.Type, ev.Data)
				return true
			}
		})
	}
}

// persistReview 把本次评审序列化后写入 store；缓存写失败仅记日志，不影响响应。
// 用 context.Background() 与请求生命周期解耦：写缓存时客户端可能已断开。
func persistReview(s store.Store, pr gh.PullRequest, summary string, risks, suggestions json.RawMessage) {
	payload, err := json.Marshal(cachedPayload{
		Summary:     summary,
		Risks:       risks,
		Suggestions: suggestions,
	})
	if err != nil {
		slog.Error("cache marshal", "err", err)
		return
	}
	rec := &store.Record{
		ID:       store.NewID(),
		Owner:    pr.Owner,
		Repo:     pr.Repo,
		PRNumber: pr.Number,
		HeadSHA:  pr.HeadSHA,
		Payload:  payload,
	}
	if err := s.Put(context.Background(), rec); err != nil {
		slog.Error("cache put", "err", err, "owner", pr.Owner, "repo", pr.Repo, "pr", pr.Number)
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

