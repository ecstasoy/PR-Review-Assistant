package review

import (
	"context"
	"errors"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
)

// Risk risks_done 事件的 payload
type Risk struct {
	File     string `json:"file"`
	Line     int    `json:"line,omitempty"`
	Severity string `json:"severity"` // high | medium | low
	Category string `json:"category"` // bug | security | perf | style | other
	Reason   string `json:"reason"`
}

// RisksStage 渲染 docs/prompts/risks.tmpl，吃 L1+L2+L3，按 JSON Schema 输出 []Risk。
// 低 severity 默认折叠。PR #8 实现。
type RisksStage struct{}

// Name 实现 Stage
func (RisksStage) Name() string { return "risks" }

// Run 实现 Stage
func (RisksStage) Run(ctx context.Context, c prctx.Context, p llm.Provider) (<-chan Event, error) {
	return nil, errors.New("RisksStage.Run: not implemented yet (PR #8)")
}
