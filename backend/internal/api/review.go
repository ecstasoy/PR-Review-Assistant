package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/review"
)

// PostReview 接收 { url } 入参，并行跑 summary + risks 两阶段，返 JSON。
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

		pCtx := buildContext(pr)

		// 两阶段并行；summary 失败 → 500，risks 失败 → 仅 log warn，返空数组
		var (
			wg         sync.WaitGroup
			summary    string
			summaryErr string
			risks      []review.Risk
			risksErr   string
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			summary, summaryErr = runSummary(ctx, pCtx, d.Provider)
		}()
		go func() {
			defer wg.Done()
			risks, risksErr = runRisks(ctx, pCtx, d.Provider)
		}()
		wg.Wait()

		if summaryErr != "" {
			slog.Error("summary stage", "err", summaryErr)
			c.JSON(500, gin.H{"error": "summary failed", "detail": summaryErr})
			return
		}
		if risksErr != "" {
			slog.Warn("risks stage failed; returning empty risks", "err", risksErr)
		}
		if risks == nil {
			risks = []review.Risk{} // 保证 JSON 输出 [] 而非 null
		}

		c.JSON(200, gin.H{
			"id":       pr.HeadSHA,
			"owner":    pr.Owner,
			"repo":     pr.Repo,
			"pr":       pr.Number,
			"url":      url,
			"head_sha": pr.HeadSHA,
			"title":    pr.Title,
			"summary":  summary,
			"risks":    risks,
		})
	}
}

// runSummary SummaryStage，把所有 delta 拼成 markdown 文本
func runSummary(ctx context.Context, c prctx.Context, p llm.Provider) (text, errMsg string) {
	events, err := (review.SummaryStage{}).Run(ctx, c, p)
	if err != nil {
		return "", err.Error()
	}
	var buf strings.Builder
	for ev := range events {
		switch ev.Type {
		case "summary_delta":
			var pl struct {
				Delta string `json:"delta"`
			}
			_ = json.Unmarshal(ev.Data, &pl)
			buf.WriteString(pl.Delta)
		case "error":
			var pl struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(ev.Data, &pl)
			errMsg = pl.Message
		}
	}
	return buf.String(), errMsg
}

// runRisks RisksStage，把 risks_done 事件 unmarshal 成 []Risk
func runRisks(ctx context.Context, c prctx.Context, p llm.Provider) (risks []review.Risk, errMsg string) {
	events, err := (review.RisksStage{}).Run(ctx, c, p)
	if err != nil {
		return nil, err.Error()
	}
	for ev := range events {
		switch ev.Type {
		case "risks_done":
			_ = json.Unmarshal(ev.Data, &risks)
		case "error":
			var pl struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(ev.Data, &pl)
			errMsg = pl.Message
		}
	}
	return risks, errMsg
}

// buildContext 把 PullRequest 拍平成最简 prctx.Context（仅 L1 + L2 patch）。
// 真正的多层裁剪 + L3 注入由后续 PR 在 prctx.Builder 实现。
func buildContext(pr gh.PullRequest) prctx.Context {
	var l1 strings.Builder
	fmt.Fprintf(&l1, "仓库: %s/%s#%d\n", pr.Owner, pr.Repo, pr.Number)
	fmt.Fprintf(&l1, "标题: %s\n", pr.Title)
	if pr.Body != "" {
		fmt.Fprintf(&l1, "正文:\n%s\n", pr.Body)
	}
	fmt.Fprintf(&l1, "改动 %d 个文件\n", len(pr.Files))

	files := make([]prctx.FileContext, 0, len(pr.Files))
	for _, f := range pr.Files {
		files = append(files, prctx.FileContext{
			Path:  f.Path,
			Patch: f.Patch,
		})
	}

	return prctx.Context{
		L1Meta:  l1.String(),
		L2Files: files,
	}
}
