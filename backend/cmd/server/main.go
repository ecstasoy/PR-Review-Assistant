// 后端 HTTP 服务入口：装配 slog → 加载 .env → 读配置 → 构造依赖 → Gin → 注册路由。
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/api"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/config"
	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/store"
)

func main() {
	// 全局 JSON 结构化日志
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// 尝试加载 .env（相对于当前工作目录）；生产环境一般无 .env：文件不存在时忽略，其它错误给出告警
	for _, p := range []string{".env", "backend/.env"} {
		if err := godotenv.Load(p); err == nil {
			slog.Info("loaded env file", "path", p)
			break
		} else if !os.IsNotExist(err) {
			slog.Warn("failed to load env file", "path", p, "err", err)
		}
	}

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

// buildDeps 按配置选 Provider；明示用户意图与最终走向，缺 key 显式 warn 而非静默降级。
// Store 打开失败仅 warn，handler 已 nil-safe；缓存 + 历史功能降级停用而非整个服务挂。
func buildDeps(cfg config.Config) api.Deps {
	provider := pickProvider(cfg)
	deps := api.Deps{
		Fetcher:  gh.NewRealFetcher(cfg.GithubToken),
		Provider: provider,
		Builder:  prctx.NewLayeredBuilder(),
	}
	// 确保 SQLite 文件的父目录存在；sqlite3 自己不会建目录
	if cfg.SQLitePath != ":memory:" {
		if dir := filepath.Dir(cfg.SQLitePath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				slog.Warn("ensure sqlite dir failed", "dir", dir, "err", err)
			}
		}
	}
	if s, err := store.NewSQLiteStore(cfg.SQLitePath); err != nil {
		slog.Warn("open sqlite store failed; cache + history disabled", "path", cfg.SQLitePath, "err", err)
	} else {
		slog.Info("store ready", "path", cfg.SQLitePath)
		deps.Store = s
	}
	return deps
}

func pickProvider(cfg config.Config) llm.Provider {
	// LLM_PROVIDER 不区分大小写，容忍 "OpenAI" / "OPENAI" / "openai" 等写法
	switch strings.ToLower(strings.TrimSpace(cfg.LLMProvider)) {
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			slog.Warn("LLM_PROVIDER=openai 但 OPENAI_API_KEY 未设，降级到 mock；请检查 .env 或 shell 环境变量")
			slog.Info("llm provider", "type", "mock", "reason", "missing key")
			return llm.NewMockProvider()
		}
		slog.Info("llm provider", "type", "openai", "base", cfg.OpenAIBaseURL, "model", cfg.LLMModel)
		return llm.NewOpenAIProvider(cfg.OpenAIBaseURL, cfg.OpenAIAPIKey, cfg.LLMModel)
	case "mock", "":
		slog.Info("llm provider", "type", "mock")
		return llm.NewMockProvider()
	default:
		slog.Warn("未知 LLM_PROVIDER 值，降级到 mock", "value", cfg.LLMProvider)
		return llm.NewMockProvider()
	}
}
