package api

import "github.com/gin-gonic/gin"

// PostReview 接收 { url } 入参，调起总结阶段，返 JSON 结果。
// 真实现下个 commit 接通；SSE 升级是独立 PR。
func PostReview(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(501, gin.H{"error": "not implemented"})
	}
}
