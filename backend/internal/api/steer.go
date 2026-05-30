package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/review"
)

// 允许 steer 重跑的 stage 集合。summary 重跑收益低（成本高 + 用户主要追问的是风险点 / 建议），暂不开放。
var allowedSteerStages = map[string]review.Stage{
	"risks":       review.RisksStage{},
	"suggestions": review.SuggestionsStage{},
}

// PostSteer POST /api/review/:id/steer
//
// 用户在会话视图底部 SteerComposer 输入引导（如 "重点看并发安全"），从 cached payload 重建 prctx，
// 把引导文本 prepend 到 L1Meta，重新跑指定 stage（默认 risks）；SSE 推 `steered_risks_done` /
// `steered_suggestions_done` 帧 + 终止 done。前端把结果合并到现有 state（替换而非追加）。
//
// 不重新 fetch GitHub：cached files 已经够；首轮的 L3 conventions 没存 cache，重跑时 L3 为空。
// 这是一个 v2 折中：steer 质量略弱于首轮但响应快且不消耗 GitHub API 配额。
func PostSteer(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if d.Store == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "steer disabled: store not configured"})
			return
		}

		id := c.Param("id")
		var body struct {
			Text  string `json:"text"`
			Stage string `json:"stage"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		text := strings.TrimSpace(body.Text)
		if text == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
			return
		}
		stageKey := body.Stage
		if stageKey == "" {
			stageKey = "risks"
		}
		stage, ok := allowedSteerStages[stageKey]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "stage must be one of: risks, suggestions"})
			return
		}

		ctx := c.Request.Context()
		rec, err := d.Store.GetByID(ctx, id)
		if err != nil {
			slog.Error("steer get review", "err", err, "id", id)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if rec == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
			return
		}
		var p cachedPayload
		if err := json.Unmarshal(rec.Payload, &p); err != nil {
			slog.Error("steer payload unmarshal", "err", err, "id", id)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "corrupted cache payload"})
			return
		}

		// 从 cached payload 反向重建一个 PullRequest 给 LayeredBuilder
		pr := gh.PullRequest{
			Owner:     rec.Owner,
			Repo:      rec.Repo,
			Number:    rec.PRNumber,
			HeadSHA:   rec.HeadSHA,
			Title:     p.Title,
			Author:    p.Author,
			State:     p.State,
			Labels:    p.Labels,
			BaseRef:   p.BaseRef,
			HeadRef:   p.HeadRef,
			CreatedAt: p.PRCreatedAt,
			Stats:     p.Stats,
			CI:        p.CI,
			Checks:    p.Checks,
			Files:     p.Files,
			// Conventions 没存 cache；L3 跑出来为空，stage prompt 会少这一段
		}

		builder := d.Builder
		if builder == nil {
			builder = prctx.NewLayeredBuilder()
		}
		pCtx, err := builder.Build(pr)
		if err != nil {
			slog.Error("steer build prctx", "err", err, "id", id)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "build context failed"})
			return
		}

		// 把用户引导 prepend 到 L1Meta，让 prompt 第一眼就看见引导意图
		pCtx.L1Meta = fmt.Sprintf("【用户引导】%s\n\n%s", text, pCtx.L1Meta)

		// SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		// info 帧：让前端会话视图能立即插一步「用户引导」running 状态
		writeSSE(c.Writer, "info", map[string]string{
			"message": fmt.Sprintf("正在按引导重跑 %s 阶段…", stageKey),
			"stage":   stageKey,
		})
		c.Writer.Flush()

		events, err := stage.Run(ctx, pCtx, d.Provider)
		if err != nil {
			writeSSE(c.Writer, "error", map[string]string{
				"stage":   "steer:" + stageKey,
				"message": err.Error(),
			})
			writeSSE(c.Writer, "done", map[string]any{})
			return
		}

		// 把 stage 原帧名（risks_done / suggestions_done）转译成 steered_* 推给前端
		// 错误 / done 帧直传
		c.Stream(func(w io.Writer) bool {
			select {
			case <-ctx.Done():
				return false
			case ev, ok := <-events:
				if !ok {
					writeSSERaw(w, "done", json.RawMessage(`{}`))
					return false
				}
				eventType := ev.Type
				switch ev.Type {
				case "risks_done":
					eventType = "steered_risks_done"
				case "suggestions_done":
					eventType = "steered_suggestions_done"
				case "done":
					return true // 跳过 stage 内部 terminal done；上层统一发
				}
				writeSSERaw(w, eventType, ev.Data)
				return true
			}
		})
	}
}

