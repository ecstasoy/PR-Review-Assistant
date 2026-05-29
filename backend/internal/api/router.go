package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/api/middleware"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
)

// Deps 是路由 + handler 需要的所有依赖。
// 在 main 构造一次，向下注入。
type Deps struct {
	Fetcher  github.Fetcher
	Provider llm.Provider
	Builder  prctx.Builder
}

// Register 挂载 /api 路由组。
func Register(r *gin.Engine, d Deps) {
	g := r.Group("/api")
	g.Use(middleware.AuthCtx())

	g.GET("/health", Health)
	g.POST("/review", PostReview(d))
	g.GET("/reviews", ListReviews(d))
	g.GET("/reviews/:id", GetReview(d))
}
