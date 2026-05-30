package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitConfig 单端点限流参数。
type RateLimitConfig struct {
	// RPS 平均速率（每秒允许的请求数）；token bucket 的 refill rate
	RPS float64
	// Burst 突发上限；首次请求可一次性消耗到这个数
	Burst int
}

// 默认配置：对昂贵端点（/review、/steer 烧 LLM）保守限频，
// 对读路径（/reviews / /reviews/:id）放宽，/health 不限。
// 真部署阶段从 env 注入；v3 切 Redis 计数后可跨实例共享。
var (
	// ExpensiveDefault 评审 + 引导：每分钟 ~12 次（10s/次），burst 5
	ExpensiveDefault = RateLimitConfig{RPS: 0.2, Burst: 5}
	// ReadDefault 列表 + 详情：每秒 ~5 次，burst 20
	ReadDefault = RateLimitConfig{RPS: 5, Burst: 20}
)

// RateLimit 返回一个 per-IP token bucket 限流中间件。
// 超限返 429 + Retry-After header（秒）。
//
// 实现说明：
//   - 用内存 sync.Map[ip] *rate.Limiter；进程重启即清空
//   - 不做 TTL 清理；长尾 IP 累积内存可控（每个 limiter ~100 字节）
//   - 多实例部署时各实例独立计数（v3 切 Redis 后跨实例共享）
//   - ClientIP 走 gin 的解析：优先 X-Forwarded-For 链中的客户端 IP，
//     必须配合反代正确设 trusted proxies；裸跑时取 RemoteAddr
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	if cfg.RPS <= 0 {
		panic("RateLimit: RPS must be > 0")
	}
	var limiters sync.Map // map[string]*rate.Limiter
	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := getLimiter(&limiters, ip, cfg)
		r := l.Reserve()
		if delay := r.Delay(); delay > 0 {
			r.Cancel()
			retryAfter := int(math.Ceil(delay.Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "请求过于频繁，请稍后再试",
				"retry_after": retryAfter,
			})
			return
		}
		c.Next()
	}
}

// getLimiter 取或建一个 ip 对应的 limiter；
// LoadOrStore 保证并发首次 set 只有一个胜出。
func getLimiter(m *sync.Map, ip string, cfg RateLimitConfig) *rate.Limiter {
	if v, ok := m.Load(ip); ok {
		return v.(*rate.Limiter)
	}
	l := rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Burst)
	actual, _ := m.LoadOrStore(ip, l)
	return actual.(*rate.Limiter)
}

