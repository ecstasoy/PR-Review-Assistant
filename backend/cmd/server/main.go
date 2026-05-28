// 后端 HTTP 服务入口：装配 slog → 读配置 → Gin → 注册路由。
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/api"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/config"
)

func main() {
	// 全局 JSON 结构化日志
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.MustLoad()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	api.RegisterRoutes(r)

	addr := fmt.Sprintf(":%d", cfg.Port)
	slog.Info("server starting", "addr", addr, "provider", cfg.LLMProvider)
	if err := r.Run(addr); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}
