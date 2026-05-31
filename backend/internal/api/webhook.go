package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/oauth"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/review"
)

// WebhookPR GitHub pull_request webhook payload 字段子集
// 只取触发评审需要的；忽略 base/head ref 等大段
type WebhookPR struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		HTMLURL string `json:"html_url"`
		Title   string `json:"title"`
		HeadSHA string `json:"-"` // 后端 Fetch 时拿
	} `json:"pull_request"`
	Repository struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	} `json:"repository"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// WebhookGitHub POST /api/webhook/github
//
// 安全：HMAC-SHA256 校验 X-Hub-Signature-256（用 GITHUB_APP_WEBHOOK_SECRET）
// 事件路由：仅 pull_request.opened 触发；其它 204
// 异步：respond 200 立刻；review 在 goroutine 跑（10-30s，超 GitHub 10s 重试阈值）
//
// 幂等：同 (owner, repo, pr, head_sha) 已有 review → 跳过；仅 push 通知
func WebhookGitHub(d Deps, webhookSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "read body: " + err.Error()})
			return
		}

		// HMAC 校验：sha256=<hex>
		sig := c.GetHeader("X-Hub-Signature-256")
		if webhookSecret == "" {
			slog.Warn("webhook: WEBHOOK_SECRET 未配；拒绝以防伪造")
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook secret not configured"})
			return
		}
		if !verifyHMAC(sig, body, []byte(webhookSecret)) {
			slog.Warn("webhook: HMAC mismatch", "sig_prefix", safePrefix(sig))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "signature mismatch"})
			return
		}

		event := c.GetHeader("X-GitHub-Event")
		if event != "pull_request" {
			// ping / installation / 其它事件先忽略
			c.JSON(http.StatusOK, gin.H{"ok": true, "ignored": event})
			return
		}

		var p WebhookPR
		if err := json.Unmarshal(body, &p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parse: " + err.Error()})
			return
		}

		// 仅 opened 触发（后续 v3 可扩 synchronize / reopened）
		if p.Action != "opened" {
			c.JSON(http.StatusOK, gin.H{"ok": true, "ignored_action": p.Action})
			return
		}
		if p.PullRequest.HTMLURL == "" || p.Repository.Owner.Login == "" || p.Repository.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "payload missing required fields"})
			return
		}

		slog.Info("webhook: pull_request.opened received",
			"owner", p.Repository.Owner.Login, "repo", p.Repository.Name,
			"pr", p.Number, "sender", p.Sender.Login, "installation", p.Installation.ID)

		// 立刻响应 GitHub（避免 10s 超时重试），review 走 goroutine
		c.JSON(http.StatusAccepted, gin.H{"ok": true, "queued": true})

		go runWebhookReview(d, webhookReviewArgs{
			PrURL:          p.PullRequest.HTMLURL,
			Owner:          p.Repository.Owner.Login,
			Repo:           p.Repository.Name,
			Number:         p.Number,
			Title:          p.PullRequest.Title,
			InstallationID: p.Installation.ID,
			SenderLogin:    p.Sender.Login,
		})
	}
}

type webhookReviewArgs struct {
	PrURL          string
	Owner          string
	Repo           string
	Number         int
	Title          string
	InstallationID int64
	SenderLogin    string
}

// runWebhookReview 后台跑评审 + 持久化 + push 通知 + post bot review
// 任一步失败 log 但不重试（demo 简化；prod 加 queue + retry）
func runWebhookReview(d Deps, args webhookReviewArgs) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pr, err := d.Fetcher.Fetch(ctx, args.PrURL)
	if err != nil {
		slog.Error("webhook: fetch PR failed", "err", err, "url", args.PrURL)
		return
	}

	// 幂等检查：同 (owner, repo, pr, head_sha) 已评 → 跳过
	if d.Store != nil {
		if rec, _ := d.Store.Get(ctx, pr.Owner, pr.Repo, pr.Number, pr.HeadSHA); rec != nil {
			slog.Info("webhook: cache hit, skipping re-review", "review_id", rec.ID)
			if args.SenderLogin != "" {
				PushNotification(ctx, d.Cache, args.SenderLogin, Notification{
					ID:       newNotifID(),
					ReviewID: rec.ID,
					Owner:    pr.Owner, Repo: pr.Repo, PR: pr.Number, Title: args.Title,
					Source: "webhook",
				})
			}
			return
		}
	}

	// 同步索引（同 manual review 路径）
	if d.Indexer != nil {
		indexPRChunks(ctx, d.Indexer, pr)
	}

	builder := d.Builder
	if builder == nil {
		builder = prctx.NewLayeredBuilder()
	}
	pCtx, err := builder.Build(ctx, pr)
	if err != nil {
		slog.Error("webhook: build prctx failed", "err", err)
		return
	}
	budget := toBudgetPayload(pCtx.BudgetReport)

	ctxByStage := buildPerStageContexts(ctx, builder, pr, pCtx)
	merged := mergeStages(ctx, ctxByStage, d.Provider)

	var (
		summaryBuf      strings.Builder
		risksData       json.RawMessage
		suggestionsData json.RawMessage
		errSeen         bool
	)
	for ev := range merged {
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
			errSeen = true
		}
	}
	if errSeen || risksData == nil || suggestionsData == nil {
		slog.Warn("webhook: review missing stage data; skip persist + bot review")
		return
	}

	var reviewID string
	if d.Store != nil {
		reviewID = persistReview(d.Store, pr, summaryBuf.String(), risksData, suggestionsData, budget, "webhook")
	}

	// Push bot review 回 PR（用 installation token）
	if reviewID != "" && d.OAuthClient != nil && d.OAuthClient.AppID != 0 && len(d.OAuthClient.PrivateKeyPEM) > 0 {
		if err := postBotReview(ctx, d.OAuthClient, args.InstallationID, pr, reviewID, summaryBuf.String(), suggestionsData); err != nil {
			slog.Warn("webhook: post bot review failed", "err", err)
		}
	} else {
		slog.Info("webhook: skip bot review (App ID / private key not configured)")
	}

	// Push 通知给 sender（评审完成）
	if args.SenderLogin != "" && reviewID != "" {
		PushNotification(ctx, d.Cache, args.SenderLogin, Notification{
			ID:       newNotifID(),
			ReviewID: reviewID,
			Owner:    pr.Owner, Repo: pr.Repo, PR: pr.Number, Title: args.Title,
			Source: "webhook",
		})
	}
	slog.Info("webhook: review pipeline done",
		"owner", pr.Owner, "repo", pr.Repo, "pr", pr.Number, "review_id", reviewID)
}

// postBotReview 用 installation token 以 App bot 身份发完整 review
// summary 在 body + lgtm.com 链接；每条 suggestion 作 inline comment（含 ```suggestion 块）
//
// 整段流程：
//  1. AppJWT 签名
//  2. 换 installation token
//  3. 解 suggestions JSON → 跳过缺 file/line 的（fork 文件名映射 issue）
//  4. PostPRReview 一次性发完整 review
func postBotReview(
	ctx context.Context,
	c *oauth.Client,
	installationID int64,
	pr gh.PullRequest,
	reviewID, summary string,
	suggRaw json.RawMessage,
) error {
	if installationID == 0 {
		return fmt.Errorf("installation ID missing in webhook payload")
	}
	jwt, err := oauth.AppJWT(c.AppID, c.PrivateKeyPEM)
	if err != nil {
		return fmt.Errorf("app jwt: %w", err)
	}
	tok, err := c.GetInstallationToken(ctx, jwt, installationID)
	if err != nil {
		return fmt.Errorf("installation token: %w", err)
	}

	var suggestions []review.Suggestion
	if len(suggRaw) > 0 {
		if err := json.Unmarshal(suggRaw, &suggestions); err != nil {
			return fmt.Errorf("parse suggestions: %w", err)
		}
	}

	inline := make([]oauth.ReviewCommentInline, 0, len(suggestions))
	for _, s := range suggestions {
		if s.File == "" || s.Line <= 0 {
			continue
		}
		inline = append(inline, oauth.ReviewCommentInline{
			Path: s.File,
			Line: s.Line,
			Side: "RIGHT",
			Body: buildSuggestionCommentBody(s),
		})
	}

	body := buildBotReviewBody(summary, reviewID, len(suggestions))
	_, err = c.PostPRReview(ctx, tok.Token, pr.Owner, pr.Repo, pr.Number, pr.HeadSHA, body, inline)
	return err
}

// buildBotReviewBody 摘要 + 评审统计 + 跳 lgtm.com 链接
func buildBotReviewBody(summary, reviewID string, sgCount int) string {
	var sb strings.Builder
	sb.WriteString("## 🤖 LGTM AI 自动评审\n\n")
	if summary != "" {
		sb.WriteString(summary)
		sb.WriteString("\n\n")
	}
	sb.WriteString(fmt.Sprintf("✨ 共生成 **%d** 条修改建议，已附在 inline 评论里。点 GitHub 自带的「Apply suggestion」可一键 commit。\n\n", sgCount))
	if reviewID != "" {
		sb.WriteString(fmt.Sprintf("🔗 完整评审（含风险列表 + RAG 召回上下文）：https://lgtm-alpha.vercel.app/review/%s\n", reviewID))
	}
	sb.WriteString("\n<sub>由 LGTM 自动生成。回复 `/lgtm review` 可触发重评（v3 计划中）</sub>")
	return sb.String()
}

// verifyHMAC GitHub 签名格式 "sha256=<hex>"；timing-safe 比较
func verifyHMAC(sig string, body, secret []byte) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(sig, prefix) {
		return false
	}
	want, err := hex.DecodeString(strings.TrimPrefix(sig, prefix))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	got := mac.Sum(nil)
	return hmac.Equal(want, got)
}

func safePrefix(s string) string {
	if len(s) > 16 {
		return s[:16] + "..."
	}
	return s
}

// newNotifID 短 ulid-like id；ms 时间戳 + 随机后缀
// 不用 ulid 库避免新增依赖；用 time.Now().UnixNano() + 简短 random
func newNotifID() string {
	now := time.Now().UnixNano()
	return fmt.Sprintf("%d-%d", now, now%1_000_000)
}
