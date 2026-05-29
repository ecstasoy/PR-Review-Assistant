package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// reviewListItem /api/reviews 列表项；只含 meta，不带 payload。
type reviewListItem struct {
	ID        string `json:"id"`
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	PR        int    `json:"pr"`
	HeadSHA   string `json:"head_sha"`
	Title     string `json:"title,omitempty"`
	CreatedAt string `json:"created_at"`
}

// reviewDetail /api/reviews/:id 详情；inline 缓存 payload 的字段。
type reviewDetail struct {
	reviewListItem
	Summary     string          `json:"summary"`
	Risks       json.RawMessage `json:"risks,omitempty"`
	Suggestions json.RawMessage `json:"suggestions,omitempty"`
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
			// title 在 payload 里；解析失败不影响主路径（旧缓存可能没有该字段）
			var p cachedPayload
			if jerr := json.Unmarshal(r.Payload, &p); jerr == nil {
				it.Title = p.Title
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
				ID:        rec.ID,
				Owner:     rec.Owner,
				Repo:      rec.Repo,
				PR:        rec.PRNumber,
				HeadSHA:   rec.HeadSHA,
				Title:     p.Title,
				CreatedAt: rec.CreatedAt.UTC().Format(time.RFC3339),
			},
			Summary:     p.Summary,
			Risks:       p.Risks,
			Suggestions: p.Suggestions,
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
