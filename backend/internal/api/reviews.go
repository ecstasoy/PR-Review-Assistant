package api

import "github.com/gin-gonic/gin"

// ListReviews 返回分页的历史评审列表。
// PR #14 实现；v1 暂按 user_id IS NULL 过滤（尚未接入登录）。
func ListReviews(c *gin.Context) {
	c.JSON(501, gin.H{"error": "not implemented", "todo": "PR #14"})
}

// GetReview 按 id 返回单条缓存评审。
// PR #15 实现；支持回放与实时 SSE 跟随两种模式。
func GetReview(c *gin.Context) {
	c.JSON(501, gin.H{"error": "not implemented", "todo": "PR #15"})
}
