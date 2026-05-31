package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RepoPermission GitHub `/repos/.../collaborators/{u}/permission` 返回的 permission 字段
// 值域见 https://docs.github.com/rest/collaborators/collaborators#get-repository-permissions-for-a-user
type RepoPermission string

const (
	PermAdmin    RepoPermission = "admin"
	PermMaintain RepoPermission = "maintain"
	PermWrite    RepoPermission = "write"
	PermTriage   RepoPermission = "triage"
	PermRead     RepoPermission = "read"
	PermNone     RepoPermission = "none"
)

// CanComment 至少 triage 权限即可发 PR review comment
// 实际 GitHub API：pull_requests:write App 权限 + 用户 ≥ triage 即可
func (p RepoPermission) CanComment() bool {
	switch p {
	case PermAdmin, PermMaintain, PermWrite, PermTriage:
		return true
	}
	return false
}

// CanCommit 至少 write 权限可以 push 提交到分支
// 注意 fork PR 时需要单独 check head repo 权限（不是 base repo）；本 helper 假设 base 仓
func (p RepoPermission) CanCommit() bool {
	switch p {
	case PermAdmin, PermMaintain, PermWrite:
		return true
	}
	return false
}

// permResponse GitHub `/collaborators/{u}/permission` 返回结构
type permResponse struct {
	Permission string `json:"permission"`
	User       struct {
		Login string `json:"login"`
	} `json:"user"`
}

// GetRepoPermission 调 GitHub Collaborators API 拿当前用户对 owner/repo 的权限
// 用户对该 repo 没访问权时 GitHub 返 404 → 退化成 PermNone（不视为错误）
// 其它非 200 才返 err；这样 caller 直接判 .CanComment() 即可
func (c *Client) GetRepoPermission(ctx context.Context, accessToken, owner, repo, username string) (RepoPermission, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/collaborators/%s/permission", owner, repo, username)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PermNone, fmt.Errorf("oauth: build perm req: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := c.httpClient().Do(req)
	if err != nil {
		return PermNone, fmt.Errorf("oauth: get perm: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	switch res.StatusCode {
	case http.StatusOK:
		var p permResponse
		if err := json.Unmarshal(body, &p); err != nil {
			return PermNone, fmt.Errorf("oauth: parse perm: %w", err)
		}
		return RepoPermission(p.Permission), nil
	case http.StatusNotFound:
		// 404 = 仓库私有且用户无访问 / 仓库不存在 / 用户非 collaborator
		return PermNone, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		// token 失效 / 缺权限；返 err 让 caller 走 401 处理
		return PermNone, fmt.Errorf("oauth: perm forbidden (%d): %s", res.StatusCode, string(body))
	default:
		return PermNone, fmt.Errorf("oauth: perm %d: %s", res.StatusCode, string(body))
	}
}
