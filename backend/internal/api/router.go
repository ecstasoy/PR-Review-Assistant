package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/api/middleware"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/store"
)

// Deps 是路由 + handler 需要的所有依赖。
// 在 main 构造一次，向下注入。
// Store 可为 nil（关闭缓存 + 历史功能），handler 必须 nil-safe。
type Deps struct {
	Fetcher  github.Fetcher
	Provider llm.Provider
	Builder  prctx.Builder
	Store    store.Store
}

// Register 挂载 /api 路由组。
func Register(r *gin.Engine, d Deps) {
	g := r.Group("/api")
	g.Use(middleware.AuthCtx())

	expensive := middleware.RateLimit(middleware.ExpensiveDefault)
	read := middleware.RateLimit(middleware.ReadDefault)

	// /health 留为 liveness 别名（向后兼容 v1/v2 配置）
	g.GET("/health", Health)
	g.GET("/health/live", Health)
	g.GET("/health/ready", Readiness(d))

	g.POST("/review", expensive, PostReview(d))
	g.GET("/reviews", read, ListReviews(d))
	g.GET("/reviews/:id", read, GetReview(d))
	g.POST("/review/:id/steer", expensive, PostSteer(d))
}
