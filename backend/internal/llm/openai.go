package llm

import (
	"context"
	"errors"
)

// OpenAIProvider 调 OpenAI 兼容的 /v1/chat/completions；默认 DeepSeek。
type OpenAIProvider struct {
	BaseURL string
	APIKey  string
	Model   string
}

// NewOpenAIProvider 构造器
func NewOpenAIProvider(baseURL, apiKey, model string) *OpenAIProvider {
	return &OpenAIProvider{BaseURL: baseURL, APIKey: apiKey, Model: model}
}

// Stream 以 stream=true 调用并转发 delta。PR #3 实现真正的 HTTP + SSE 解析。
func (p *OpenAIProvider) Stream(ctx context.Context, req Request) (<-chan Chunk, error) {
	return nil, errors.New("OpenAIProvider.Stream: not implemented yet")
}
