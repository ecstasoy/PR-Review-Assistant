package api

import "github.com/gin-gonic/gin"

// Health 存活探针，用来确认后端运行
func Health(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
