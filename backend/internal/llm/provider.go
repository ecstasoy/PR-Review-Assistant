// Package llm 抽象 OpenAI 兼容的流式接口
package llm

import (
	"context"
	"encoding/json"
)

// Message 多轮对话中的一条消息。
// agent 循环需要把工具结果回灌作 role=tool 的新消息，再调一次 Stream。
type Message struct {
	Role       string     `json:"role"`                   // system / user / assistant / tool
	Content    string     `json:"content,omitempty"`      // assistant 用空 + ToolCalls 表示发起 tool 调用
	Name       string     `json:"name,omitempty"`         // role=tool 时是 function name
	ToolCallID string     `json:"tool_call_id,omitempty"` // role=tool 时引用的 assistant tool_call.id
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // role=assistant 发起的工具调用
}

// ToolSpec OpenAI function calling 风格的工具描述；与 agent.ToolSpec 同形状。
// 放在 llm 包让 Provider 接口不依赖 agent；agent 包通过 Tool.Spec() 返回这个。
type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}

// ToolCall LLM 决定调用的工具实例；流式累积完后 emit。
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`      // 来自 function.name
	Arguments string `json:"arguments"` // raw JSON string（按 schema 由调用方解析）
}

// Request 统一调用参数。
//
// 两种模式：
//   - 旧（System + User）：单轮 prompt，向后兼容 v1/v2 stage 调用
//   - 新（Messages）：多轮对话；Tools 非空时启用 tool calling；agent loop 用
//
// Messages 非空时优先用；System / User 则忽略。
type Request struct {
	// 兼容字段（旧的单轮调用）
	System string
	User   string

	// 多轮对话 + tool calling（新）
	Messages []Message
	Tools    []ToolSpec

	Temperature float32
	Model       string  // 空串表示 Provider 默认模型
	JSONSchema  *Schema // 非 nil 时强制结构化输出（与 Tools 互斥；同时设以 Tools 优先）
}

// Schema 约束 LLM 输出
type Schema struct {
	Name string
	JSON json.RawMessage
}

// Chunk 一帧流式增量
type Chunk struct {
	Text      string     // assistant content 增量
	ToolCalls []ToolCall // 一轮完整 tool_calls 解析完 emit（非增量；index 已聚合）
	Done      bool
	Err       error
}

// Provider 流式补全
type Provider interface {
	Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}
