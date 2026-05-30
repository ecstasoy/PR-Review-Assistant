package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Health 旧 /health 端点 = liveness（无依赖检查）。保留以维持兼容。
// 用法：容器编排 liveness probe、Docker HEALTHCHECK、UptimeRobot
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness 检查下游依赖是否可用。任意 fail → 503，让 LB / k8s 摘流。
// 检查项：
//   - store：如果 Deps.Store 非 nil，调 Ping
//   - 未来：Cache.Ping / LLM provider 健康（不接 LLM —— 第三方慢且不稳，
//     readiness 自身要快、要确定）
//
// 超时阈值 1s：避免阻塞编排器；DB 一秒 ping 不上视为不健康
func Readiness(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		checks := map[string]string{}
		allHealthy := true

		if d.Store != nil {
			if err := d.Store.Ping(ctx); err != nil {
				slog.Error("store ping failed", "err", err)
				checks["store"] = "fail"
				allHealthy = false
			} else {
				checks["store"] = "ok"
			}
		} else {
			checks["store"] = "disabled"
		}

		status := "ready"
		code := http.StatusOK
		if !allHealthy {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}
		c.JSON(code, gin.H{"status": status, "checks": checks})
	}
}
