package api

import "github.com/gin-gonic/gin"

// PostReview 接收 { url } 入参，对外以 SSE 推送 summary / risks / suggestions 三阶段事件。
// PR #4 先打通一次性 JSON 返回；PR #12 升级为真正的 SSE 流。
func PostReview(c *gin.Context) {
	c.JSON(501, gin.H{"error": "not implemented", "todo": "PR #4"})
}
