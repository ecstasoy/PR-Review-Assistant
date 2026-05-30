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

	// /health 不限：容器健康检查 / k8s liveness 高频探测，限了会被误杀
	g.GET("/health", Health)

	// 昂贵端点（烧 LLM）单独建限流器
	expensive := middleware.RateLimit(middleware.ExpensiveDefault)
	g.POST("/review", expensive, PostReview(d))
	g.POST("/review/:id/steer", expensive, PostSteer(d))

	// 读路径放宽：列表 / 详情仅查 SQLite，单实例千 QPS 不痛
	read := middleware.RateLimit(middleware.ReadDefault)
	g.GET("/reviews", read, ListReviews(d))
	g.GET("/reviews/:id", read, GetReview(d))
}
