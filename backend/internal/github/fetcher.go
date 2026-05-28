// Package github 抓 PR：meta + 改动文件 + 仓库约定文件。
package github

import "context"

// PullRequest 一次抓取的完整 PR 快照。
type PullRequest struct {
	Owner       string
	Repo        string
	Number      int
	HeadSHA     string
	Title       string
	Body        string
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
