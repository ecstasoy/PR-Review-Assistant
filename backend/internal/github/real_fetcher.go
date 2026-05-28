package github

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v66/github"
)

// RealFetcher 通过 go-github 调 GitHub REST。
type RealFetcher struct {
	client *gh.Client
}

// NewRealFetcher 构造 RealFetcher
func NewRealFetcher(token string) *RealFetcher {
	c := gh.NewClient(nil)
	if token != "" {
		c = c.WithAuthToken(token)
	}
	return &RealFetcher{client: c}
}

// Fetch 实现 Fetcher。
// 拉 PR meta + 改动文件列表
// 文件列表上限 100；超出由 PR #7 在 prctx 层裁剪。
// Conventions 留空，PR #11 实施。
func (f *RealFetcher) Fetch(ctx context.Context, rawURL string) (PullRequest, error) {
	owner, repo, number, err := ParseURL(rawURL)
	if err != nil {
		return PullRequest{}, err
	}

	pr, _, err := f.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return PullRequest{}, fmt.Errorf("get pull request: %w", err)
	}

	files, _, err := f.client.PullRequests.ListFiles(ctx, owner, repo, number, &gh.ListOptions{PerPage: 100})
	if err != nil {
		return PullRequest{}, fmt.Errorf("list pull request files: %w", err)
	}

	out := PullRequest{
		Owner:   owner,
		Repo:    repo,
		Number:  number,
		HeadSHA: pr.GetHead().GetSHA(),
		Title:   pr.GetTitle(),
		Body:    pr.GetBody(),
	}
	for _, file := range files {
		out.Files = append(out.Files, File{
			Path:      file.GetFilename(),
			Status:    file.GetStatus(),
			Patch:     file.GetPatch(),
			Additions: file.GetAdditions(),
			Deletions: file.GetDeletions(),
		})
	}
	return out, nil
}
