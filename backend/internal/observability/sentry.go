// Package observability 装错误跟踪 / metrics / tracing 等"非业务必需但很有用"的接入。
// v1 全部 noop；按 env 条件开启（如 Sentry DSN / OTLP endpoint 非空时才初始化）。
package observability

import (
	"context"
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

// SentryConfig 接入 Sentry 所需的最小集
type SentryConfig struct {
	DSN         string // 空时整个 Sentry 不启用
	Environment string // dev / staging / prod；空时默认 "development"
	Release     string // 版本 tag 或 git sha；空时 Sentry 自己估
}

// InitSentry 在 DSN 非空时初始化 Sentry SDK；返回 cleanup（main defer 调用）。
// DSN 空时返回 noop cleanup —— 让 main 接线无条件 defer 即可，不用判 nil。
func InitSentry(cfg SentryConfig) (cleanup func(), err error) {
	noop := func() {}
	if cfg.DSN == "" {
		slog.Info("sentry disabled (SENTRY_DSN empty)")
		return noop, nil
	}
	env := cfg.Environment
	if env == "" {
		env = "development"
	}
	err = sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      env,
		Release:          cfg.Release,
		EnableTracing:    false, // v1 只跟错误，不开 distributed tracing（开销大）
		AttachStacktrace: true,
		// 不抓含 ctx.Value 的敏感字段；prod 部署后按需调严
		SendDefaultPII: false,
	})
	if err != nil {
		slog.Error("sentry init failed; running without it", "err", err)
		return noop, err
	}
	slog.Info("sentry enabled", "environment", env)
	// cleanup：main 退出前 flush 2 秒，避免最后一批事件丢
	return func() { sentry.Flush(2 * time.Second) }, nil
}

// Recover 把任意 panic 上报 Sentry；defer 调用即可。
// 调用方应该自己继续 panic 或处理，本函数仅 capture 不吞错。
func Recover(ctx context.Context) {
	if r := recover(); r != nil {
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub()
		}
		hub.Recover(r)
		hub.Flush(2 * time.Second)
		panic(r) // 继续向上传播，让上层（如 gin Recovery）决定怎么处理
	}
}

// CaptureError 把 err 上报 Sentry；ctx 可带 hub 信息。
// 业务代码遇到非预期错误（如 LLM provider 故障、cache 故障）时手动调用。
func CaptureError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	hub.CaptureException(err)
}
