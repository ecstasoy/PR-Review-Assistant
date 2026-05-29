package api

import "github.com/gin-gonic/gin"

// ListReviews 返回分页的历史评审列表。
func ListReviews(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(501, gin.H{"error": "not implemented"})
	}
}

// GetReview 按 id 返回单条缓存评审。
func GetReview(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(501, gin.H{"error": "not implemented"})
	}
}
