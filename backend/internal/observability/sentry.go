package observability

import (
	"time"

	"github.com/getsentry/sentry-go"
)

type SentryConfig struct {
	DSN         string
	Environment string
}

func InitSentry(cfg SentryConfig) (func(), error) {
	if cfg.DSN == "" {
		return func() {}, nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		AttachStacktrace: true,
		EnableTracing:    false,
		SendDefaultPII:   false,
	})
	if err != nil {
		return func() {}, err
	}

	return func() {
		sentry.Flush(2 * time.Second)
	}, nil
}

func Recover(ctx *sentry.Hub) {
	if r := recover(); r != nil {
		if ctx != nil {
			ctx.Recover(r)
		}
		panic(r)
	}
}

func CaptureError(ctx *sentry.Hub, err error) {
	if err == nil || ctx == nil {
		return
	}
	ctx.CaptureException(err)
}

func CurrentHub() *sentry.Hub {
	return sentry.CurrentHub()
}
