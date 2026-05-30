package observability

import (
	"context"
	"errors"
	"testing"
)

func TestInitSentry_NoDSNReturnsNoopCleanup(t *testing.T) {
	cleanup, err := InitSentry(SentryConfig{DSN: ""})
	if err != nil {
		t.Errorf("unexpected init err with empty DSN: %v", err)
	}
	if cleanup == nil {
		t.Fatalf("cleanup should not be nil even when DSN empty")
	}
	// noop 调用安全
	cleanup()
}

func TestInitSentry_InvalidDSN(t *testing.T) {
	// 非法 DSN 应返 err，但 cleanup 仍非 nil（noop）
	cleanup, err := InitSentry(SentryConfig{DSN: "not-a-url"})
	if err == nil {
		t.Errorf("expected init err with bad DSN")
	}
	if cleanup == nil {
		t.Fatalf("cleanup should still be non-nil on init error")
	}
	cleanup()
}

func TestCaptureError_NilSafe(t *testing.T) {
	// nil error 应该是 noop，不 panic
	CaptureError(context.Background(), nil)
}

func TestCaptureError_RealError(t *testing.T) {
	// Sentry 没初始化时 CaptureException 走全局 noop hub，不 panic
	CaptureError(context.Background(), errors.New("test error"))
}
