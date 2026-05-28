package middleware

import "github.com/gin-gonic/gin"

// AuthCtx 是 v1 的空操作鉴权中间件。
//
// v2 会替换为 OAuth / session 校验，解析出调用者的 user id。
// Handler 始终从 gin.Context 读取 userID，因此替换实现时不需要改动 handler 签名。
func AuthCtx() gin.HandlerFunc {
	return func(c *gin.Context) {
		var userID *string // v1 永远为 nil
		c.Set("userID", userID)
		c.Next()
	}
}
