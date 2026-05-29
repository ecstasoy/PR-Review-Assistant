package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
)

// fakeFetcher 让测试注入预设的 PR 数据或错误。
type fakeFetcher struct {
	pr  gh.PullRequest
	err error
}

func (f fakeFetcher) Fetch(_ context.Context, _ string) (gh.PullRequest, error) {
	return f.pr, f.err
}

func newTestRouter(t *testing.T, deps Deps) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, deps)
	return r
}

func postJSON(t *testing.T, r http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPostReview_Success(t *testing.T) {
	mock := llm.NewMockProvider()
	mock.Reply = "这是一个测试评审 总结"

	deps := Deps{
		Fetcher: fakeFetcher{
			pr: gh.PullRequest{
				Owner:   "golang",
				Repo:    "go",
				Number:  42,
				HeadSHA: "deadbeef",
				Title:   "fix race",
				Body:    "fixes #123",
				Files: []gh.File{
					{Path: "scanner.go", Status: "modified", Patch: "@@ -1 +1 @@\n-old\n+new"},
				},
			},
		},
		Provider: mock,
	}

	r := newTestRouter(t, deps)
	w := postJSON(t, r, "/api/review", map[string]string{"url": "https://github.com/golang/go/pull/42"})

	if w.Code != 200 {
		t.Fatalf("status=%d, want 200; body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if resp["id"] != "deadbeef" {
		t.Errorf("id=%v, want deadbeef", resp["id"])
	}
	if resp["title"] != "fix race" {
		t.Errorf("title=%v", resp["title"])
	}
	summary, _ := resp["summary"].(string)
	if !strings.Contains(summary, "测试") || !strings.Contains(summary, "评审") {
		t.Errorf("summary 应含 mock 推送的关键词，得到 %q", summary)
	}
}

func TestPostReview_InvalidJSON(t *testing.T) {
	deps := Deps{Fetcher: fakeFetcher{}, Provider: llm.NewMockProvider()}
	r := newTestRouter(t, deps)

	req := httptest.NewRequest(http.MethodPost, "/api/review", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("status=%d, want 400", w.Code)
	}
}

func TestPostReview_EmptyURL(t *testing.T) {
	deps := Deps{Fetcher: fakeFetcher{}, Provider: llm.NewMockProvider()}
	r := newTestRouter(t, deps)
	w := postJSON(t, r, "/api/review", map[string]string{"url": ""})

	if w.Code != 400 {
		t.Errorf("status=%d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestPostReview_InvalidURL(t *testing.T) {
	deps := Deps{
		Fetcher:  fakeFetcher{err: gh.ErrInvalidPRURL},
		Provider: llm.NewMockProvider(),
	}
	r := newTestRouter(t, deps)
	w := postJSON(t, r, "/api/review", map[string]string{"url": "https://gitlab.com/foo/bar/pull/1"})

	if w.Code != 400 {
		t.Errorf("status=%d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestPostReview_FetcherError(t *testing.T) {
	deps := Deps{
		Fetcher:  fakeFetcher{err: errors.New("github 504")},
		Provider: llm.NewMockProvider(),
	}
	r := newTestRouter(t, deps)
	w := postJSON(t, r, "/api/review", map[string]string{"url": "https://github.com/o/r/pull/1"})

	if w.Code != 502 {
		t.Errorf("status=%d, want 502", w.Code)
	}
}
