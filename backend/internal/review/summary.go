package review

import (
	"context"
	"errors"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
)

// SummaryStage 渲染 docs/prompts/summary.tmpl，L1 + L3，输出 markdown 总结
// PR #4 实现。
type SummaryStage struct{}

// Name 实现 Stage
func (SummaryStage) Name() string { return "summary" }

// Run 实现 Stage
func (SummaryStage) Run(ctx context.Context, c prctx.Context, p llm.Provider) (<-chan Event, error) {
	return nil, errors.New("SummaryStage.Run: not implemented yet (PR #4)")
}
