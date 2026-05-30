package observability

import "testing"

func TestInitTracer_EmptyEndpoint_NoopCleanup(t *testing.T) {
	cleanup, err := InitTracer(OTelConfig{Endpoint: ""})
	if err != nil {
		t.Errorf("empty endpoint should not err, got %v", err)
	}
	if cleanup == nil {
		t.Fatalf("cleanup should not be nil even when disabled")
	}
	cleanup()
}

func TestStripScheme(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"http://collector:4318", "collector:4318"},
		{"https://collector:4318", "collector:4318"},
		{"collector:4318", "collector:4318"}, // 已经无 scheme 时不动
		{"", ""},
	}
	for _, c := range cases {
		if got := stripScheme(c.in); got != c.want {
			t.Errorf("stripScheme(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestInitTracer_WithEndpoint_NoOpCleanupOnUnreachable(t *testing.T) {
	// 给一个非法 endpoint：exporter 会尝试连接，但 BatchSpanProcessor 会缓冲、Shutdown 时再 flush。
	// 这里只验证 cleanup 不 panic、不挂死（5s 超时已设）。
	cleanup, _ := InitTracer(OTelConfig{
		Endpoint: "127.0.0.1:1", // 故意端口 1，拒绝连接
		Insecure: true,
	})
	if cleanup == nil {
		t.Fatalf("cleanup should not be nil")
	}
	cleanup() // 不应卡死或 panic
}
