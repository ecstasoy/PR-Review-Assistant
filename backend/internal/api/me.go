package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MeResponse /api/me 返回字段；未登录时 Authenticated=false 其它字段空
// 后续 G5+G6a PR 会扩展 perms[]（针对当前查看 PR 的权限）
type MeResponse struct {
	Authenticated bool   `json:"authenticated"`
	Login         string `json:"login,omitempty"`
	UserID        int64  `json:"user_id,omitempty"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	Name          string `json:"name,omitempty"`
}

// GetMe GET /api/me
// 让前端检查登录态：未登录 → 显示 Sign in 按钮；已登录 → 显示头像 + 登出
func GetMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		s := CurrentSession(c)
		if s == nil {
			c.JSON(http.StatusOK, MeResponse{Authenticated: false})
			return
		}
		c.JSON(http.StatusOK, MeResponse{
			Authenticated: true,
			Login:         s.Login,
			UserID:        s.UserID,
			AvatarURL:     s.AvatarURL,
			Name:          s.Name,
		})
	}
}
