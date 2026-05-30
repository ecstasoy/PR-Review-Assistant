// Package agent 是 v2 工具调用 / agent 循环的入口。
// v1 每个 review stage 只调一次 LLM；v2 可把某个 stage 换成 Agent.Run
// 实现"LLM -> 选工具 -> 跑工具 -> 回灌结果 -> 再问"的循环。
package agent

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
)

// ToolSpec OpenAI function-calling 风格的工具描述。
// 直接复用 llm 包的同形状定义：Registry.Specs() 可直传 llm.Request.Tools。
type ToolSpec = llm.ToolSpec

// Tool 一项可调用能力。
type Tool interface {
	Spec() ToolSpec
	Run(ctx context.Context, args json.RawMessage) (string, error)
}

// Registry 按名字存放 Tool。
type Registry struct {
	tools map[string]Tool
}

// NewRegistry 空注册表。
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 按 Spec().Name 注册;同名覆盖。
func (r *Registry) Register(t Tool) {
	r.tools[t.Spec().Name] = t
}

// Lookup 按名查 Tool。
func (r *Registry) Lookup(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Specs 返回所有 ToolSpec，供 prompt 注入。
func (r *Registry) Specs() []ToolSpec {
	specs := make([]ToolSpec, 0, len(r.tools))
	for _, t := range r.tools {
		specs = append(specs, t.Spec())
	}
	return specs
}

// Result Agent 循环结束后的最终输出。
type Result struct {
	Output string
	Steps  int
}

// Agent 一个工具调用循环。
type Agent struct {
	Provider llm.Provider
	Tools    *Registry
	MaxSteps int
}

// Run 跑循环。v2 实现：LLM -> 工具调用 -> 把结果作为 user message 回灌 -> 再调 LLM；最多 MaxSteps 轮。
func (a *Agent) Run(ctx context.Context, req llm.Request) (Result, error) {
	return Result{}, errors.New("agent.Agent.Run: not implemented yet (reserved for v2)")
}
