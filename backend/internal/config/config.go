package config

import (
	"log/slog"
	"os"

	"github.com/caarlos0/env/v11"
)

// Config 读取并加载所有环境变量中的配置。
// 默认值保证 `go run ./cmd/server` 无需任何配置即可启动
// LLM 可降级到 mock
type Config struct {
	Port int `env:"PORT" envDefault:"8080"`

	GithubToken string `env:"GITHUB_TOKEN"`

	LLMProvider   string `env:"LLM_PROVIDER" envDefault:"mock"`
	OpenAIAPIKey  string `env:"OPENAI_API_KEY"`
	OpenAIBaseURL string `env:"OPENAI_BASE_URL" envDefault:"https://api.deepseek.com"`
	LLMModel      string `env:"LLM_MODEL" envDefault:"deepseek-chat"`

	SQLitePath string `env:"SQLITE_PATH" envDefault:"./data/reviews.db"`

	// v3 真部署：PostgresURL 非空时 store 切到 PostgresStore；否则继续走 SQLite。
	// 业务代码（handler / router）零变化，差异在 main 接线时按 cfg.PostgresURL != "" 二选一。
	// 形如：postgres://user:pass@host:5432/dbname?sslmode=disable
	PostgresURL string `env:"POSTGRES_URL"`

	// v3 真部署：RedisURL 非空时 cache 切到 RedisCache；否则走 MemoryCache。
	// 形如：redis://:password@host:6379/0
	// 用途：rate limit 计数跨实例共享 / SSE session 状态 / 未来 RAG 索引队列
	RedisURL string `env:"REDIS_URL"`

	// v3 OAuth / GitHub App：空时回退到旧 PAT 路径（GithubToken）
	GithubAppID         int64  `env:"GITHUB_APP_ID"`
	GithubAppPrivateKey string `env:"GITHUB_APP_PRIVATE_KEY"`    // PEM 内容或文件路径
	GithubAppWebhookSec string `env:"GITHUB_APP_WEBHOOK_SECRET"` // webhook 签名校验
	GithubOAuthClientID string `env:"GITHUB_OAUTH_CLIENT_ID"`
	GithubOAuthSecret   string `env:"GITHUB_OAUTH_CLIENT_SECRET"`

	// v3 可观测性：Sentry / OTLP；空 = 不启用
	SentryDSN    string `env:"SENTRY_DSN"`
	OTLPEndpoint string `env:"OTLP_ENDPOINT"`                        // OTel collector 地址，如 http://otel-collector:4318
	Environment  string `env:"ENVIRONMENT" envDefault:"development"` // dev / staging / prod，给 Sentry tag 用

	// TrustedProxies 信任的反代 IP/CIDR 列表（逗号分隔）；
	// 用于 c.ClientIP() 正确解析 X-Forwarded-For。
	// 空字符串（默认）表示不信任任何代理，直接取 RemoteAddr。
	TrustedProxies string `env:"TRUSTED_PROXIES" envDefault:""`
}

// MustLoad 解析环境变量；失败则打印错误并退出进程。
func MustLoad() Config {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		slog.Error("config load", "err", err)
		os.Exit(1)
	}
	return cfg
}
