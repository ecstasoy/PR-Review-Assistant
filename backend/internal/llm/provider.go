// Package llm 抽象 OpenAI 兼容的流式接口
package llm

import (
	"context"
	"encoding/json"
)

// Request 统一调用参数
type Request struct {
	System      string
	User        string
	Temperature float32
	Model       string  // 空串表示 Provider 默认模型
	JSONSchema  *Schema // 非 nil 时强制结构化输出
}

// Schema 约束 LLM 输出
type Schema struct {
	Name string
	JSON json.RawMessage
}

// Chunk 一帧流式增量
type Chunk struct {
	Text string
	Done bool
	Err  error
}

// Provider 流式补全
type Provider interface {
	Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}
