// Package github 抓 PR：meta + 改动文件 + 仓库约定文件。
package github

import (
	"context"
	"time"
)

// PR 状态常量；State 字段值取自 GitHub API（open/closed），merged 在 Fetcher 层合并 merged 标志后归类。
const (
	StateOpen   = "open"
	StateClosed = "closed"
	StateMerged = "merged"
)

// Stats PR 体量统计（来自 GitHub API 的 pulls.Get 响应，无需额外请求）。
type Stats struct {
	Files     int // changed_files
	Additions int
	Deletions int
	Commits   int
	Comments  int // PR 评论数（不含 review comments）
}

// PullRequest 一次抓取的完整 PR 快照。
type PullRequest struct {
	Owner   string
	Repo    string
	Number  int
	HeadSHA string
	Title   string
	Body    string

	// PR meta：用于落地页"最近评审"卡 / 评审顶栏 / 历史表格 / 会话视图分支 chip
	Author    string    // GitHub login，可空（极少数情况）
	State     string    // open / closed / merged
	Labels    []string  // 标签名列表，按 API 返回顺序
	BaseRef   string    // base 分支名（如 "main"）
	HeadRef   string    // head 分支名（如 "fix/shard-eviction-race"）
	CreatedAt time.Time // PR 创建时间，UTC
	Stats     Stats

	Files       []File
	Conventions Conventions
}

// File 一个改动文件。
type File struct {
	Path      string
	Status    string // added | modified | removed | renamed
	Patch     string // diff hunk
	FullText  string // 可选；prctx.Builder 按预算决定是否拉全文
	Additions int
	Deletions int
}

// Conventions 仓库级约定文件，作为 L3 上下文。
type Conventions struct {
	Readme       string
	Contributing string
	AgentDocs    string // CLAUDE.md 或 AGENTS.md，取存在的那个
}

// Fetcher 按 URL 拉 PR 快照。PR #2 实现 google/go-github 版。
type Fetcher interface {
	Fetch(ctx context.Context, url string) (PullRequest, error)
}
