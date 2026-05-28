package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/api/middleware"
)

// RegisterRoutes /api 路由组
func RegisterRoutes(r *gin.Engine) {
	g := r.Group("/api")
	g.Use(middleware.AuthCtx())

	g.GET("/health", Health)
	g.POST("/review", PostReview)
	g.GET("/reviews", ListReviews)
	g.GET("/reviews/:id", GetReview)
}
