package review

import (
	"context"
	"errors"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
)

// Suggestion suggestions_done 事件的 payload
type Suggestion struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Type       string `json:"type"` // bug | style | perf | security
	Suggestion string `json:"suggestion"`
}

// SuggestionsStage 渲染 docs/prompts/suggestions.tmpl，L1+L2，按 JSON Schema 输出 []Suggestion。
// PR #9 实现。
type SuggestionsStage struct{}

// Name 实现 Stage
func (SuggestionsStage) Name() string { return "suggestions" }

// Run 实现 Stage
func (SuggestionsStage) Run(ctx context.Context, c prctx.Context, p llm.Provider) (<-chan Event, error) {
	return nil, errors.New("SuggestionsStage.Run: not implemented yet (PR #9)")
}
