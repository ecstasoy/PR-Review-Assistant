package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/agent"
	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/review"
)

// 允许 steer 重跑的 stage 集合。summary 重跑收益低（成本高 + 用户主要追问的是风险点 / 建议），暂不开放。
var allowedSteerStages = map[string]review.Stage{
	"risks":       review.RisksStage{},
	"suggestions": review.SuggestionsStage{},
}

// buildAgentUserPrompt 把 prctx.Context 装到 agent 的 user prompt 里。
// 与 stage 模板的核心差异：不要 JSON 输出指令；显式提示 L4 RAG 段是跨 PR 上下文。
// L2 不内联（patch 体积大）—— agent 可调 read_file 按需取；这里只给 L1Meta（含文件名列表）和 L4 召回。
func buildAgentUserPrompt(pr gh.PullRequest, userQuery string, pCtx prctx.Context) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "PR：%s/%s#%d（%s）\n\n", pr.Owner, pr.Repo, pr.Number, pr.Title)
	fmt.Fprintf(&sb, "用户引导：%s\n\n", userQuery)
	sb.WriteString("## PR 元信息\n")
	sb.WriteString(pCtx.L1Meta)
	if pCtx.L3Conventions != "" {
		sb.WriteString("\n\n## 项目约定\n")
		sb.WriteString(pCtx.L3Conventions)
	}
	if len(pCtx.L4References) > 0 {
		sb.WriteString("\n\n## 相关代码（跨文件 RAG 召回；可能来自本 PR 或之前评过的同 repo PR）\n")
		for _, r := range pCtx.L4References {
			origin := r.Reason
			if r.PRNumber > 0 {
				origin = fmt.Sprintf("来自 PR #%d · %s", r.PRNumber, r.Reason)
			}
			fmt.Fprintf(&sb, "\n**%s**（%s）\n```\n%s\n```\n", r.File, origin, r.Snippet)
		}
	}
	return sb.String()
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
			Mode  string `json:"mode"` // "stage"（默认，重跑 risks/suggestions）/ "agent"（跑 ReAct loop + 工具调用）
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
		mode := body.Mode
		if mode == "" {
			mode = "stage"
		}
		if mode != "stage" && mode != "agent" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "mode must be one of: stage, agent"})
			return
		}
		// stage 字段在 mode=stage 时必填；agent 模式忽略
		stageKey := body.Stage
		if stageKey == "" {
			stageKey = "risks"
		}
		stage, stageOK := allowedSteerStages[stageKey]
		if mode == "stage" && !stageOK {
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
		// P2: 用用户输入作 RAG query，让追问 / steer 召回更对题（而非默认的 PR 元信息）
		pCtx, err := builder.BuildWith(ctx, pr, prctx.BuildOptions{RAGQuery: text})
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

		// mode=agent：跑 ReAct loop + 工具调用；通过 WireAgentSSE 自动推 tool_call_start/done 帧
		if mode == "agent" {
			writeSSE(c.Writer, "info", map[string]string{
				"message": "Agent 启动：可调用 read_file / list_dir / grep_patches 工具深挖",
				"stage":   "agent",
			})
			c.Writer.Flush()

			reg := agent.NewRegistry()
			agent.RegisterDefaults(reg, p.Files)

			a := &agent.Agent{
				Provider: d.Provider,
				Tools:    reg,
				MaxSteps: 5,
			}
			WireAgentSSE(a, c.Writer)

			sysPrompt := "你是 code reviewer agent。可调用 read_file / list_dir / grep_patches 三个工具深挖 PR 改动。" +
				"分析时先想清楚要看哪些文件 / grep 什么模式，再调工具。" +
				"prompt 末尾「相关代码」段是 RAG 召回的同 repo 内容（可能来自本 PR 或之前评过的同 repo PR），优先据此回答跨文件问题。" +
				"最后用一段简洁中文文字总结你的发现（不要 JSON）。"
			userPrompt := buildAgentUserPrompt(pr, text, pCtx)

			result, err := a.Run(ctx, llm.Request{System: sysPrompt, User: userPrompt})
			if err != nil {
				slog.Warn("steer agent run failed", "err", err, "steps", result.Steps)
				writeSSE(c.Writer, "error", map[string]string{
					"stage":   "agent",
					"message": err.Error(),
				})
			}
			// 把 agent 最终输出作 info 帧推给前端（v1 简化：不试图 parse 成 risks/suggestions JSON）
			if result.Output != "" {
				writeSSE(c.Writer, "info", map[string]string{
					"message": fmt.Sprintf("Agent 完成（%d 步）：%s", result.Steps, result.Output),
					"stage":   "agent",
				})
			}
			writeSSE(c.Writer, "done", map[string]any{})
			c.Writer.Flush()
			return
		}

		// 默认 mode=stage：重跑 risks 或 suggestions stage
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

