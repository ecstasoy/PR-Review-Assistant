package observability

import (
	"errors"
	"testing"
)

func TestInitSentry_NoDSNReturnsNoopCleanup(t *testing.T) {
	cleanup, err := InitSentry(SentryConfig{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cleanup == nil {
		t.Fatal("expected non-nil cleanup")
	}
	cleanup()
}

func TestInitSentry_InvalidDSN(t *testing.T) {
	cleanup, err := InitSentry(SentryConfig{DSN: "://bad-dsn"})
	if err == nil {
		t.Fatal("expected error for invalid DSN")
	}
	if cleanup == nil {
		t.Fatal("expected non-nil cleanup on error")
	}
	cleanup()
}

func TestCaptureError_NilSafe(t *testing.T) {
	hub := CurrentHub()
	CaptureError(hub, nil)
}

func TestCaptureError_RealError(t *testing.T) {
	hub := CurrentHub()
	CaptureError(hub, errors.New("boom"))
}

func TestRecover_Repanics(t *testing.T) {
	hub := CurrentHub()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic to be rethrown")
		}
	}()

	func() {
		defer Recover(hub)
		panic("boom")
	}()
}

func TestRecover_NoPanic(t *testing.T) {
	hub := CurrentHub()
	func() {
		defer Recover(hub)
	}()
}

func TestRecover_NilHubRepanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic to be rethrown")
		}
	}()

	func() {
		defer Recover(nil)
		panic("boom")
	}()
}

func TestCurrentHub_NonNil(t *testing.T) {
	if CurrentHub() == nil {
		t.Fatal("expected non-nil hub")
	}
}

func TestCaptureError_NilHubSafe(t *testing.T) {
	CaptureError(nil, errors.New("boom"))
}
