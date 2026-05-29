package github

import (
	"context"
	"errors"
	"net/http"

	gh "github.com/google/go-github/v66/github"
)

// maxConventionFileSize 单个约定文件字节上限；超出按 byte 截断。
// Builder 之后还会按 token 预算二次截断，这里只是第一道防线。
const maxConventionFileSize = 16384

// agentDocCandidates 约定文档候选名，按优先级排序：先命中者胜出。
var agentDocCandidates = []string{"CLAUDE.md", "AGENTS.md"}

// fetchConventions 拉仓库根目录的约定文件，缺失静默跳过。
// ref 通常传 PR HeadSHA，确保看到 PR 自身视角的约定（PR 可能新增/修改这些文件）。
func fetchConventions(ctx context.Context, client *gh.Client, owner, repo, ref string) (Conventions, error) {
	opts := &gh.RepositoryContentGetOptions{Ref: ref}

	readme, err := fetchFileContent(ctx, client, owner, repo, "README.md", opts)
	if err != nil {
		return Conventions{}, err
	}
	contributing, err := fetchFileContent(ctx, client, owner, repo, "CONTRIBUTING.md", opts)
	if err != nil {
		return Conventions{}, err
	}

	agentDocs := ""
	for _, name := range agentDocCandidates {
		content, err := fetchFileContent(ctx, client, owner, repo, name, opts)
		if err != nil {
			return Conventions{}, err
		}
		if content != "" {
			agentDocs = content
			break
		}
	}

	return Conventions{
		Readme:       readme,
		Contributing: contributing,
		AgentDocs:    agentDocs,
	}, nil
}

// fetchFileContent 拉单个文件文本；404 返回空串；超长按字节截断。
func fetchFileContent(ctx context.Context, client *gh.Client, owner, repo, path string, opts *gh.RepositoryContentGetOptions) (string, error) {
	fileContent, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return "", nil
		}
		var errResp *gh.ErrorResponse
		if errors.As(err, &errResp) && errResp.Response != nil && errResp.Response.StatusCode == http.StatusNotFound {
			return "", nil
		}
		return "", err
	}
	if fileContent == nil {
		// 路径是目录而非文件，跳过
		return "", nil
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return "", err
	}
	if len(content) > maxConventionFileSize {
		cut := 0
		for i := range content {
			if i > maxConventionFileSize {
				break
			}
			cut = i
		}
		content = content[:cut] + "\n...[truncated]"
	}
	return content, nil
}
