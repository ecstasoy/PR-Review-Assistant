package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// graphQLURL GitHub GraphQL v4 endpoint
const graphQLURL = "https://api.github.com/graphql"

// graphQLRequest 通用 GraphQL POST body
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse 解 errors[] 出来：GraphQL 200 + body errors 也算失败
type graphQLResponse[T any] struct {
	Data   T `json:"data"`
	Errors []struct {
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
		Path    []any  `json:"path,omitempty"`
	} `json:"errors,omitempty"`
}

// doGraphQL 通用 POST 调；T 是 caller 期望的 data shape
// errors 字段非空时即使 HTTP 200 也返 err
func doGraphQL[T any](ctx context.Context, c *Client, token string, req graphQLRequest) (T, error) {
	var zero T
	raw, _ := json.Marshal(req)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, graphQLURL, bytes.NewReader(raw))
	if err != nil {
		return zero, fmt.Errorf("graphql: build req: %w", err)
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Accept", "application/json")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := c.httpClient().Do(r)
	if err != nil {
		return zero, fmt.Errorf("graphql: do: %w", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("graphql: status %d: %s", res.StatusCode, string(body))
	}
	var parsed graphQLResponse[T]
	if err := json.Unmarshal(body, &parsed); err != nil {
		return zero, fmt.Errorf("graphql: parse: %w (body=%s)", err, string(body))
	}
	if len(parsed.Errors) > 0 {
		msgs := make([]string, len(parsed.Errors))
		for i, e := range parsed.Errors {
			msgs[i] = e.Message
		}
		return zero, fmt.Errorf("graphql: %s", joinStrings(msgs, "; "))
	}
	return parsed.Data, nil
}

func joinStrings(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

// FindReviewThreadID 查 PR 的 review threads，按 comment.databaseId 找到包含该 comment 的 thread id
// 用 last:30 覆盖刚 POST 的 comment（如果你的 PR 评论 > 30 条且新发的不在最后 30 里，这里会找不到 → 返 err）
// 实务足够：刚发完立刻找，几乎一定在最后 1-3 个 thread 里
func (c *Client) FindReviewThreadID(
	ctx context.Context,
	accessToken, owner, repo string,
	prNumber int,
	commentDatabaseID int64,
) (string, error) {
	type threadNode struct {
		ID       string `json:"id"`
		Comments struct {
			Nodes []struct {
				DatabaseID int64 `json:"databaseId"`
			} `json:"nodes"`
		} `json:"comments"`
	}
	type queryData struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					Nodes []threadNode `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	query := `query($owner:String!,$repo:String!,$pr:Int!){
        repository(owner:$owner,name:$repo){
            pullRequest(number:$pr){
                reviewThreads(last:30){
                    nodes{id comments(first:5){nodes{databaseId}}}
                }
            }
        }
    }`
	data, err := doGraphQL[queryData](ctx, c, accessToken, graphQLRequest{
		Query: query,
		Variables: map[string]any{
			"owner": owner, "repo": repo, "pr": prNumber,
		},
	})
	if err != nil {
		return "", err
	}
	for _, th := range data.Repository.PullRequest.ReviewThreads.Nodes {
		for _, com := range th.Comments.Nodes {
			if com.DatabaseID == commentDatabaseID {
				return th.ID, nil
			}
		}
	}
	return "", errors.New("graphql: thread for comment not found in last 30 threads")
}

// ApplySuggestionResult 提交成功的结果；包含新 commit oid 让前端给用户看
type ApplySuggestionResult struct {
	CommitOID  string
	IsResolved bool
}

// ApplyReviewThreadSuggestion 调 applyPullRequestReviewThreadSuggestion mutation
// 等价于用户在 GitHub PR UI 上点 "Apply suggestion"：会生成一条 commit push 到 PR head ref
//
// 失败常见原因：
//   - fork PR 且 maintainer_can_modify=false → 403（caller 翻译成 "fork PR 未开放编辑"）
//   - thread 已 resolved → 422
//   - 用户对 head repo 无 push 权限 → 403
func (c *Client) ApplyReviewThreadSuggestion(
	ctx context.Context,
	accessToken, threadID string,
) (*ApplySuggestionResult, error) {
	type mutationData struct {
		ApplyPullRequestReviewThreadSuggestion struct {
			PullRequest struct {
				HeadRef struct {
					Target struct {
						OID string `json:"oid"`
					} `json:"target"`
				} `json:"headRef"`
			} `json:"pullRequest"`
			PullRequestReviewThread struct {
				IsResolved bool `json:"isResolved"`
			} `json:"pullRequestReviewThread"`
		} `json:"applyPullRequestReviewThreadSuggestion"`
	}
	mutation := `mutation($id:ID!){
        applyPullRequestReviewThreadSuggestion(input:{pullRequestReviewThreadId:$id}){
            pullRequest{headRef{target{... on Commit{oid}}}}
            pullRequestReviewThread{isResolved}
        }
    }`
	data, err := doGraphQL[mutationData](ctx, c, accessToken, graphQLRequest{
		Query:     mutation,
		Variables: map[string]any{"id": threadID},
	})
	if err != nil {
		return nil, err
	}
	return &ApplySuggestionResult{
		CommitOID:  data.ApplyPullRequestReviewThreadSuggestion.PullRequest.HeadRef.Target.OID,
		IsResolved: data.ApplyPullRequestReviewThreadSuggestion.PullRequestReviewThread.IsResolved,
	}, nil
}
