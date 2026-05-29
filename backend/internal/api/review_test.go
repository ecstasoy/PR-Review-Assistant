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

// dualMockProvider 按 req.JSONSchema 是否非 nil 切换 reply：
// JSONSchema==nil（summary 阶段）→ textReply
// JSONSchema!=nil（risks 阶段）→ jsonReply
type dualMockProvider struct {
	textReply string
	jsonReply string
}

func (d dualMockProvider) Stream(_ context.Context, req llm.Request) (<-chan llm.Chunk, error) {
	reply := d.textReply
	if req.JSONSchema != nil {
		reply = d.jsonReply
	}
	ch := make(chan llm.Chunk, 8)
	go func() {
		defer close(ch)
		for _, w := range strings.Fields(reply) {
			ch <- llm.Chunk{Text: w + " "}
		}
		ch <- llm.Chunk{Done: true}
	}()
	return ch, nil
}

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

	// mock 推非 JSON 文本，risks 阶段必失败 → 返空数组（不能是 null）
	risks, ok := resp["risks"].([]any)
	if !ok {
		t.Errorf("risks 应是数组，得到 %T (%v)", resp["risks"], resp["risks"])
	}
	if len(risks) != 0 {
		t.Errorf("mock provider 下 risks 应为空，得到 %d 条", len(risks))
	}
}

func TestPostReview_SuccessWithRisks(t *testing.T) {
	deps := Deps{
		Fetcher: fakeFetcher{
			pr: gh.PullRequest{
				Owner:   "golang",
				Repo:    "go",
				Number:  42,
				HeadSHA: "deadbeef",
				Title:   "fix race",
				Files: []gh.File{
					{Path: "scanner.go", Status: "modified", Patch: "@@ -1 +1 @@"},
				},
			},
		},
		Provider: dualMockProvider{
			textReply: "这是一段总结",
			jsonReply: `{"risks":[{"file":"scanner.go","line":42,"severity":"high","category":"bug","confidence":0.92,"reason":"竞态"}]}`,
		},
	}

	r := newTestRouter(t, deps)
	w := postJSON(t, r, "/api/review", map[string]string{"url": "https://github.com/golang/go/pull/42"})

	if w.Code != 200 {
		t.Fatalf("status=%d, want 200; body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Summary string `json:"summary"`
		Risks   []struct {
			File       string  `json:"file"`
			Line       int     `json:"line"`
			Severity   string  `json:"severity"`
			Confidence float32 `json:"confidence"`
		} `json:"risks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if !strings.Contains(resp.Summary, "总结") {
		t.Errorf("summary 字段错: %q", resp.Summary)
	}
	if len(resp.Risks) != 1 {
		t.Fatalf("risks 数量 = %d，期望 1", len(resp.Risks))
	}
	r0 := resp.Risks[0]
	if r0.File != "scanner.go" || r0.Severity != "high" || r0.Line != 42 {
		t.Errorf("risks[0] 字段错: %+v", r0)
	}
	if r0.Confidence < 0.91 || r0.Confidence > 0.93 {
		t.Errorf("Confidence = %v，期望 ~0.92", r0.Confidence)
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
