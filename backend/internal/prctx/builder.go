// Package prctx 构建分层 prompt 上下文（L1 meta / L2 文件 / L3 约定 / L4 检索）。
// 目录名避开标准库 context；包名 prctx。
package prctx

import (
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/index"
)

// Context 结构化 prompt 输入。
type Context struct {
	L1Meta        string            // PR 标题 + body + per-file 统计
	L2Files       []FileContext     // per-file patch + 可选全文
	L3Conventions string            // README / CONTRIBUTING / agent docs 拼接
	L4References  []index.Reference // v1 永远 nil；v2 RAG 填充
	BudgetReport  BudgetReport
}

// FileContext 单文件 patch + 可选全文。
type FileContext struct {
	Path     string
	Patch    string
	FullText string // 被预算丢弃时为空
}

// Builder 把 PullRequest 转成 Context。
// 实现侧构造时接收 github.Fetcher 和 index.Retriever（v1 注入 NoopRetriever）。
type Builder interface {
	Build(pr github.PullRequest) (Context, error)
}
