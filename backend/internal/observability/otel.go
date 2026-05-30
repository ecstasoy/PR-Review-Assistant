package observability

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// OTelConfig 接入 OpenTelemetry trace 的最小集
type OTelConfig struct {
	Endpoint    string // OTLP HTTP collector 地址（如 http://otel-collector:4318）；空时整个 OTel 不启用
	Environment string // dev / staging / prod
	ServiceName string // resource.service.name；空时默认 "pr-review-assistant"
	Insecure    bool   // 跳 TLS 验证（dev / 同 cluster 内 collector 用）
}

// InitTracer 在 Endpoint 非空时初始化全局 TracerProvider；返回 cleanup（main defer 调用）。
// Endpoint 空时返回 noop cleanup —— main 无条件 defer 即可。
//
// 默认配置：
//   - 用 OTLP HTTP exporter（gRPC 多一层依赖；HTTP 通用够用）
//   - BatchSpanProcessor（生产 collector 友好；不是每 span 立刻 push）
//   - resource 标 service.name + deployment.environment
func InitTracer(cfg OTelConfig) (cleanup func(), err error) {
	noop := func() {}
	endpoint := stripScheme(cfg.Endpoint)
	if endpoint == "" {
		slog.Info("otel disabled (OTLP_ENDPOINT empty)")
		return noop, nil
	}
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "pr-review-assistant"
	}
	env := cfg.Environment
	if env == "" {
		env = "development"
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
	if hasHTTPSScheme(cfg.Endpoint) {
		cfg.Insecure = false
	} else if hasHTTPPlaintextScheme(cfg.Endpoint) {
		cfg.Insecure = true
	}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	exporter, err := otlptracehttp.New(context.Background(), opts...)
	if err != nil {
		slog.Error("otel exporter init failed; running without it", "err", err)
		return noop, err
	}

	customResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.DeploymentEnvironment(env),
	)
	res, mergeErr := resource.Merge(resource.Default(), customResource)
	if mergeErr != nil {
		slog.Warn("otel resource merge failed; using custom resource only", "err", mergeErr)
		res = customResource
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	slog.Info("otel enabled", "endpoint", endpoint, "service", serviceName, "environment", env)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			slog.Warn("otel tracer provider shutdown error", "err", err)
		}
	}, nil
}

// stripScheme 去掉 URL 的 http:// / https:// 前缀；otlptracehttp 的 WithEndpoint 只要 host:port 形态。
// 用 Insecure 选项区分协议；写法和 OpenTelemetry SDK 约定一致。
func stripScheme(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	for _, prefix := range []string{"http://", "https://"} {
		if len(endpoint) >= len(prefix) && strings.EqualFold(endpoint[:len(prefix)], prefix) {
			if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
				return parsed.Host
			}
			endpoint = endpoint[len(prefix):]
			break
		}
	}
	if idx := strings.IndexAny(endpoint, "/?#"); idx >= 0 {
		return endpoint[:idx]
	}
	return endpoint
}

func hasHTTPPlaintextScheme(endpoint string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(endpoint)), "http://")
}

func hasHTTPSScheme(endpoint string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(endpoint)), "https://")
}
