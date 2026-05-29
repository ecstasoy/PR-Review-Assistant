package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/review"
)

// PostReview 接收 { url } 入参，跑总结阶段并返 JSON 结果。
// SSE 升级是独立 PR。
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

		events, err := (review.SummaryStage{}).Run(ctx, buildContext(pr), d.Provider)
		if err != nil {
			slog.Error("summary stage", "err", err)
			c.JSON(500, gin.H{"error": "summary failed", "detail": err.Error()})
			return
		}

		var summary strings.Builder
		var streamErr string
		for ev := range events {
			switch ev.Type {
			case "summary_delta":
				var p struct {
					Delta string `json:"delta"`
				}
				_ = json.Unmarshal(ev.Data, &p)
				summary.WriteString(p.Delta)
			case "error":
				var p struct {
					Message string `json:"message"`
				}
				_ = json.Unmarshal(ev.Data, &p)
				streamErr = p.Message
			}
		}
		if streamErr != "" {
			c.JSON(500, gin.H{"error": "stream error", "detail": streamErr})
			return
		}

		c.JSON(200, gin.H{
			"id":       pr.HeadSHA,
			"owner":    pr.Owner,
			"repo":     pr.Repo,
			"pr":       pr.Number,
			"url":      url,
			"head_sha": pr.HeadSHA,
			"title":    pr.Title,
			"summary":  summary.String(),
		})
	}
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
