package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// riskCounts 按 severity 统计的风险数；落地"最近评审"卡 + 历史表格的红 / 黄 / 灰 pips 用。
type riskCounts struct {
	High   int `json:"high"`
	Medium int `json:"medium"`
	Low    int `json:"low"`
}

// reviewListItem /api/reviews 列表项；只含 meta + CI + risks 计数，不带完整 payload。
// 给落地"最近评审"卡 + /history 密集表格用，所以体积最小化。
type reviewListItem struct {
	ID         string     `json:"id"`
	Owner      string     `json:"owner"`
	Repo       string     `json:"repo"`
	PR         int        `json:"pr"`
	HeadSHA    string     `json:"head_sha"`
	Title      string     `json:"title,omitempty"`
	CreatedAt  string     `json:"created_at"`
	CI         string     `json:"ci,omitempty"`
	Lang       string     `json:"lang,omitempty"` // PR 主语言（detectPrimaryLang 的结果）；/history 语言筛选用
	Source     string     `json:"source,omitempty"` // "manual" / "webhook"；前端按此渲染 ⚡ chip
	RiskCounts riskCounts `json:"risk_counts"`
}

// reviewDetail /api/reviews/:id 详情；完整透出 cachedPayload，
// 给评审页顶栏 + 三视图渲染（review 页缓存秒回路径不再需要回 GitHub 拉 meta）。
type reviewDetail struct {
	reviewListItem
	Author      string          `json:"author,omitempty"`
	AuthorRole  string          `json:"author_role,omitempty"`
	State       string          `json:"state,omitempty"`
	Labels      []string        `json:"labels,omitempty"`
	BaseRef     string          `json:"base_ref,omitempty"`
	HeadRef     string          `json:"head_ref,omitempty"`
	PRCreatedAt time.Time       `json:"pr_created_at,omitzero"`
	Stats       gh.Stats        `json:"stats,omitzero"`
	Checks      []gh.Check      `json:"checks,omitempty"`
	Files        []gh.File            `json:"files,omitempty"` // 给 Diff 视图渲染用；list 端不返该大字段
	Summary      string               `json:"summary"`
	Risks        json.RawMessage      `json:"risks,omitempty"`
	Suggestions  json.RawMessage      `json:"suggestions,omitempty"`
	BudgetReport *budgetReportPayload `json:"budget_report,omitempty"`
}

// countRisksBySeverity 解析 risks_done event raw JSON，按 severity 分组计数。
// 解析失败返零值（容错于格式异常的旧缓存）。
func countRisksBySeverity(raw json.RawMessage) riskCounts {
	if len(raw) == 0 {
		return riskCounts{}
	}
	var items []struct {
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return riskCounts{}
	}
	var c riskCounts
	for _, it := range items {
		switch it.Severity {
		case "high":
			c.High++
		case "medium":
			c.Medium++
		case "low":
			c.Low++
		}
	}
	return c
}

// ListReviews GET /api/reviews?limit=N — 历史评审列表，按 created_at DESC。
// 未配 Store 时返 503 而非 200 空列表，让前端能区分"没有历史"和"功能未启用"。
func ListReviews(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if d.Store == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "history disabled: store not configured"})
			return
		}
		limit := parseLimit(c.Query("limit"))
		records, err := d.Store.List(c.Request.Context(), nil, limit)
		if err != nil {
			slog.Error("list reviews", "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out := make([]reviewListItem, 0, len(records))
		for _, r := range records {
			it := reviewListItem{
				ID:        r.ID,
				Owner:     r.Owner,
				Repo:      r.Repo,
				PR:        r.PRNumber,
				HeadSHA:   r.HeadSHA,
				CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
			}
			// title / ci / risks 计数从 payload 解出；解析失败保留零值（旧缓存兼容）
			var p cachedPayload
			if jerr := json.Unmarshal(r.Payload, &p); jerr == nil {
				it.Title = p.Title
				it.CI = p.CI
				it.Lang = p.Lang
				it.Source = p.Source
				it.RiskCounts = countRisksBySeverity(p.Risks)
			}
			out = append(out, it)
		}
		c.JSON(http.StatusOK, out)
	}
}

// GetReview GET /api/reviews/:id — 按 id 取详情；payload 解析失败按 500 暴露而非降级返"空 review"。
func GetReview(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if d.Store == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "history disabled: store not configured"})
			return
		}
		id := c.Param("id")
		rec, err := d.Store.GetByID(c.Request.Context(), id)
		if err != nil {
			slog.Error("get review", "err", err, "id", id)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if rec == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "review not found"})
			return
		}
		var p cachedPayload
		if err := json.Unmarshal(rec.Payload, &p); err != nil {
			slog.Error("cached payload unmarshal", "err", err, "id", id)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "corrupted cache payload"})
			return
		}
		c.JSON(http.StatusOK, reviewDetail{
			reviewListItem: reviewListItem{
				ID:         rec.ID,
				Owner:      rec.Owner,
				Repo:       rec.Repo,
				PR:         rec.PRNumber,
				HeadSHA:    rec.HeadSHA,
				Title:      p.Title,
				CreatedAt:  rec.CreatedAt.UTC().Format(time.RFC3339),
				CI:         p.CI,
				Lang:       p.Lang,
				Source:     p.Source,
				RiskCounts: countRisksBySeverity(p.Risks),
			},
			Files:        p.Files,
			Author:       p.Author,
			AuthorRole:   p.AuthorRole,
			State:        p.State,
			Labels:       p.Labels,
			BaseRef:      p.BaseRef,
			HeadRef:      p.HeadRef,
			PRCreatedAt:  p.PRCreatedAt,
			Stats:        p.Stats,
			Checks:       p.Checks,
			Summary:      p.Summary,
			Risks:        p.Risks,
			Suggestions:  p.Suggestions,
			BudgetReport: p.BudgetReport,
		})
	}
}

// parseLimit 把 query "limit" 解析为 [1, maxListLimit] 内的整数；非法或缺省返 defaultListLimit。
func parseLimit(s string) int {
	if s == "" {
		return defaultListLimit
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return defaultListLimit
	}
	if n > maxListLimit {
		return maxListLimit
	}
	return n
}
