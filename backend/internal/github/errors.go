package github

import (
	"errors"
	"net/http"

	gh "github.com/google/go-github/v66/github"
)

// ErrPRNotFound PR 不存在或仓库为私有且未给 token；handler 翻成 404。
var ErrPRNotFound = errors.New("PR not found or repository is private (set GITHUB_TOKEN)")

// ErrAccessDenied GitHub 拒绝访问，可能是 token 失效、权限不足或速率受限；handler 翻成 403。
var ErrAccessDenied = errors.New("GitHub denied access (rate limit or insufficient permissions)")

// classifyGitHubError 把 go-github ErrorResponse 翻成本包 sentinel。
// 非 ErrorResponse / 非 4xx 的错误原样返回，调用方再包一层上下文。
func classifyGitHubError(err error) error {
	var errResp *gh.ErrorResponse
	if !errors.As(err, &errResp) || errResp.Response == nil {
		return err
	}
	switch errResp.Response.StatusCode {
	case http.StatusNotFound:
		return ErrPRNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrAccessDenied
	default:
		return err
	}
}
