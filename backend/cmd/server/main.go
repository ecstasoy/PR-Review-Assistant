// 后端 HTTP 服务入口：装配 slog → 读配置 → 构造依赖 → Gin → 注册路由。
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/api"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/config"
	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
)

func main() {
	// 全局 JSON 结构化日志
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.MustLoad()
	deps := buildDeps(cfg)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	api.Register(r, deps)

	addr := fmt.Sprintf(":%d", cfg.Port)
	slog.Info("server starting", "addr", addr)
	if err := r.Run(addr); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

// buildDeps 按配置选 Provider；LLM_PROVIDER=openai 且有 key 才用真实，否则 mock。
func buildDeps(cfg config.Config) api.Deps {
	var provider llm.Provider
	if cfg.LLMProvider == "openai" && cfg.OpenAIAPIKey != "" {
		provider = llm.NewOpenAIProvider(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.LLMModel)
		slog.Info("llm provider", "type", "openai", "base", cfg.OpenAIBaseURL, "model", cfg.LLMModel)
	} else {
		provider = llm.NewMockProvider()
		slog.Info("llm provider", "type", "mock")
	}
	return api.Deps{
		Fetcher:  gh.NewRealFetcher(cfg.GithubToken),
		Provider: provider,
	}
}
