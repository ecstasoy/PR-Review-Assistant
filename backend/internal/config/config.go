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
