package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/config"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/db"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/githubclient"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/handlers"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/models"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/observability"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/openai"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/server"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	logger.Info("starting backend")

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
	}

	gormDB, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect database", "error", err)
	}

	r := gin.Default()
	if cleanup, err := observability.InitSentry(observability.SentryConfig{
		DSN:         cfg.SentryDSN,
		Environment: cfg.Environment,
	}); err != nil {
		logger.Error("failed to initialize sentry", "error", err)
	} else {
		defer cleanup()
		if cfg.SentryDSN != "" {
			r.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
		}
	}

	r.MaxMultipartMemory = 8 << 20 // 8 MiB

	gh := githubclient.New(cfg.GitHubToken)

	openaiClient := openai.New(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL)
	modelName := cfg.OpenAIModel
	if modelName == "" {
		modelName = "gpt-4.1-mini"
	}
	llmProvider := llm.NewOpenAIProvider(openaiClient, modelName)
	if cfg.LLMProvider == "mock" {
		llmProvider = llm.NewMockProvider()
	}

	repo := models.NewRepository(gormDB)
	h := handlers.NewHandler(repo, gh, llmProvider, logger)
	srv := server.New(h)
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	r.GET("/sse-test", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		for i := 0; i < 5; i++ {
			c.Writer.WriteString("data: ping\n\n")
			c.Writer.Flush()
		}
	})
	r.POST("/api/reviews", func(c *gin.Context) { srv.PostReview(c.Request.Context(), c) })
	serverAddr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		serverAddr = ":" + port
	}
	logger.Info("listening", "addr", serverAddr)
	if err := r.Run(serverAddr); err != nil {
		logger.Error("server exited", "error", err)
		os.Exit(1)
	}
	// 测试时有些直接调用 main 后继续执行；显式阻塞直到 context 被取消可避免退出过快
	<-context.Background().Done()
}
