package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

// startTestServer 起一个真实 httptest server；gin.c.Stream 在 ResponseRecorder 下会因 CloseNotify panic。
func startTestServer(t *testing.T, deps Deps) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, deps)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, srv *httptest.Server, path string, body any) (*http.Response, string) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer res.Body.Close()
	buf, _ := io.ReadAll(res.Body)
	return res, string(buf)
}

type sseFrame struct {
	Type string
	Data string
}

func parseSSE(body string) []sseFrame {
	var frames []sseFrame
	for _, raw := range strings.Split(body, "\n\n") {
		if raw == "" {
			continue
		}
		var f sseFrame
		for _, line := range strings.Split(raw, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				f.Type = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				f.Data += strings.TrimPrefix(line, "data: ")
			}
		}
		if f.Type != "" {
			frames = append(frames, f)
		}
	}
	return frames
}

// dualMockProvider 按 req.JSONSchema 是否非 nil 切换 reply。
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

func TestPostReview_Success(t *testing.T) {
	mock := llm.NewMockProvider()
	mock.Reply = "这是一个测试评审 总结"

	srv := startTestServer(t, Deps{
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
	})

	res, body := postJSON(t, srv, "/api/review", map[string]string{"url": "https://github.com/golang/go/pull/42"})

	if res.StatusCode != 200 {
		t.Fatalf("status=%d, want 200; body=%s", res.StatusCode, body)
	}
	if !strings.Contains(res.Header.Get("Content-Type"), "text/event-stream") {
		t.Errorf("Content-Type 应是 SSE，得到 %q", res.Header.Get("Content-Type"))
	}

	frames := parseSSE(body)
	if len(frames) == 0 {
		t.Fatal("SSE 没有任何帧")
	}

	if frames[0].Type != "pr" {
		t.Errorf("首帧应是 pr，得到 %q", frames[0].Type)
	}
	var prData map[string]any
	if err := json.Unmarshal([]byte(frames[0].Data), &prData); err != nil {
		t.Fatalf("解析 pr data: %v", err)
	}
	if prData["id"] != "deadbeef" {
		t.Errorf("pr.id=%v，期望 deadbeef", prData["id"])
	}
	if prData["title"] != "fix race" {
		t.Errorf("pr.title=%v", prData["title"])
	}

	var summary strings.Builder
	for _, f := range frames {
		if f.Type != "summary_delta" {
			continue
		}
		var p struct {
			Delta string `json:"delta"`
		}
		_ = json.Unmarshal([]byte(f.Data), &p)
		summary.WriteString(p.Delta)
	}
	if !strings.Contains(summary.String(), "测试") || !strings.Contains(summary.String(), "评审") {
		t.Errorf("summary 应含 mock 关键词，得到 %q", summary.String())
	}

	if frames[len(frames)-1].Type != "done" {
		t.Errorf("末帧应是 done，得到 %q", frames[len(frames)-1].Type)
	}
}

func TestPostReview_SuccessWithRisks(t *testing.T) {
	srv := startTestServer(t, Deps{
		Fetcher: fakeFetcher{
			pr: gh.PullRequest{
				Owner: "golang", Repo: "go", Number: 42, HeadSHA: "deadbeef", Title: "fix race",
				Files: []gh.File{{Path: "scanner.go", Status: "modified", Patch: "@@ -1 +1 @@"}},
			},
		},
		Provider: dualMockProvider{
			textReply: "这是一段总结",
			jsonReply: `{"risks":[{"file":"scanner.go","line":42,"severity":"high","category":"bug","confidence":0.92,"reason":"竞态"}]}`,
		},
	})

	res, body := postJSON(t, srv, "/api/review", map[string]string{"url": "https://github.com/golang/go/pull/42"})
	if res.StatusCode != 200 {
		t.Fatalf("status=%d, want 200; body=%s", res.StatusCode, body)
	}

	frames := parseSSE(body)

	var risksFrame *sseFrame
	for i := range frames {
		if frames[i].Type == "risks_done" {
			risksFrame = &frames[i]
			break
		}
	}
	if risksFrame == nil {
		t.Fatal("缺 risks_done 帧")
	}
	var risks []struct {
		File       string  `json:"file"`
		Severity   string  `json:"severity"`
		Confidence float32 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(risksFrame.Data), &risks); err != nil {
		t.Fatalf("解析 risks_done: %v", err)
	}
	if len(risks) != 1 || risks[0].Severity != "high" || risks[0].Confidence < 0.91 || risks[0].Confidence > 0.93 {
		t.Errorf("risks 字段错: %+v", risks)
	}
}

func TestPostReview_InvalidJSON(t *testing.T) {
	srv := startTestServer(t, Deps{Fetcher: fakeFetcher{}, Provider: llm.NewMockProvider()})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/review", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Errorf("status=%d, want 400", res.StatusCode)
	}
}

func TestPostReview_EmptyURL(t *testing.T) {
	srv := startTestServer(t, Deps{Fetcher: fakeFetcher{}, Provider: llm.NewMockProvider()})
	res, body := postJSON(t, srv, "/api/review", map[string]string{"url": ""})
	if res.StatusCode != 400 {
		t.Errorf("status=%d, want 400; body=%s", res.StatusCode, body)
	}
}

func TestPostReview_InvalidURL(t *testing.T) {
	srv := startTestServer(t, Deps{
		Fetcher:  fakeFetcher{err: gh.ErrInvalidPRURL},
		Provider: llm.NewMockProvider(),
	})
	res, body := postJSON(t, srv, "/api/review", map[string]string{"url": "https://gitlab.com/foo/bar/pull/1"})
	if res.StatusCode != 400 {
		t.Errorf("status=%d, want 400; body=%s", res.StatusCode, body)
	}
}

func TestPostReview_FetcherError(t *testing.T) {
	srv := startTestServer(t, Deps{
		Fetcher:  fakeFetcher{err: errors.New("github 504")},
		Provider: llm.NewMockProvider(),
	})
	res, body := postJSON(t, srv, "/api/review", map[string]string{"url": "https://github.com/o/r/pull/1"})
	if res.StatusCode != 502 {
		t.Errorf("status=%d, want 502; body=%s", res.StatusCode, body)
	}
}
