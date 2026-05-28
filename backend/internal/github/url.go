package github

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// ErrInvalidPRURL 传入的字符串不是合法 GitHub PR 链接
var ErrInvalidPRURL = errors.New("invalid GitHub PR URL")

// ParseURL 从 PR 链接解出 owner / repo / 编号
// 如：
//
//	https://github.com/owner/repo/pull/123
//	https://github.com/owner/repo/pull/123/files
//	末尾斜杠可有可无
func ParseURL(raw string) (owner, repo string, number int, err error) {
	u, parseErr := url.Parse(strings.TrimSpace(raw))
	if parseErr != nil {
		return "", "", 0, ErrInvalidPRURL
	}
	if u.Host != "github.com" {
		return "", "", 0, ErrInvalidPRURL
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	// 期望: [owner, repo, "pull", number, ...]
	if len(parts) < 4 || parts[2] != "pull" {
		return "", "", 0, ErrInvalidPRURL
	}
	n, convErr := strconv.Atoi(parts[3])
	if convErr != nil || n <= 0 {
		return "", "", 0, ErrInvalidPRURL
	}
	return parts[0], parts[1], n, nil
}
