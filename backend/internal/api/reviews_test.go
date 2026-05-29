package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/llm"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/store"
)

// seedReview 直接往 store 写一条 cached review 用于 list / get 测试
func seedReview(t *testing.T, s store.Store, owner, repo string, pr int, sha, title string, when time.Time) string {
	t.Helper()
	payload, _ := json.Marshal(cachedPayload{
		Title:       title,
		Summary:     "summary for " + title,
		Risks:       json.RawMessage(`[]`),
		Suggestions: json.RawMessage(`[]`),
	})
	id := store.NewID()
	rec := &store.Record{
		ID: id, Owner: owner, Repo: repo, PRNumber: pr, HeadSHA: sha,
		Payload: payload, CreatedAt: when,
	}
	if err := s.Put(context.Background(), rec); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return id
}

func TestListReviews_ReturnsRecordsDesc(t *testing.T) {
	s := newTestStore(t)
	seedReview(t, s, "golang", "go", 1, "sha1", "older PR", time.Unix(1000, 0))
	newerID := seedReview(t, s, "golang", "go", 2, "sha2", "newer PR", time.Unix(2000, 0))

	srv := startTestServer(t, Deps{Provider: llm.NewMockProvider(), Store: s})
	res, body := getJSON(t, srv, "/api/reviews")
	if res.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}

	var list []reviewListItem
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatalf("decode: %v body=%s", err, body)
	}
	if len(list) != 2 {
		t.Fatalf("应返 2 条，得到 %d: %s", len(list), body)
	}
	if list[0].ID != newerID {
		t.Errorf("最新一条应排首位；首条 ID=%s 期望 %s", list[0].ID, newerID)
	}
	if list[0].Title != "newer PR" {
		t.Errorf("title 应从 payload 解出，得到 %q", list[0].Title)
	}
}

func TestListReviews_LimitParam(t *testing.T) {
	s := newTestStore(t)
	for i := range 5 {
		seedReview(t, s, "o", "r", i+1, "sha", "t", time.Unix(int64(1000+i), 0))
	}
	srv := startTestServer(t, Deps{Provider: llm.NewMockProvider(), Store: s})

	res, body := getJSON(t, srv, "/api/reviews?limit=2")
	if res.StatusCode != 200 {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var list []reviewListItem
	_ = json.Unmarshal([]byte(body), &list)
	if len(list) != 2 {
		t.Errorf("limit=2 应返 2 条，得到 %d", len(list))
	}
}

func TestListReviews_NoStore_503(t *testing.T) {
	srv := startTestServer(t, Deps{Provider: llm.NewMockProvider(), Store: nil})
	res, body := getJSON(t, srv, "/api/reviews")
	if res.StatusCode != 503 {
		t.Errorf("status=%d want 503; body=%s", res.StatusCode, body)
	}
}

func TestGetReview_ReturnsDetail(t *testing.T) {
	s := newTestStore(t)
	id := seedReview(t, s, "golang", "go", 42, "deadbeef", "fix race", time.Unix(1000, 0))
	srv := startTestServer(t, Deps{Provider: llm.NewMockProvider(), Store: s})

	res, body := getJSON(t, srv, "/api/reviews/"+id)
	if res.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	var d reviewDetail
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		t.Fatalf("decode: %v body=%s", err, body)
	}
	if d.ID != id || d.PR != 42 || d.HeadSHA != "deadbeef" || d.Title != "fix race" {
		t.Errorf("meta 错: %+v", d)
	}
	if !strings.Contains(d.Summary, "fix race") {
		t.Errorf("summary 错: %q", d.Summary)
	}
	if string(d.Risks) != "[]" {
		t.Errorf("risks 应是 raw JSON [], 得到 %s", d.Risks)
	}
}

func TestGetReview_NotFound_404(t *testing.T) {
	s := newTestStore(t)
	srv := startTestServer(t, Deps{Provider: llm.NewMockProvider(), Store: s})
	res, _ := getJSON(t, srv, "/api/reviews/no-such-id")
	if res.StatusCode != 404 {
		t.Errorf("status=%d want 404", res.StatusCode)
	}
}

func TestGetReview_NoStore_503(t *testing.T) {
	srv := startTestServer(t, Deps{Provider: llm.NewMockProvider(), Store: nil})
	res, _ := getJSON(t, srv, "/api/reviews/anything")
	if res.StatusCode != 503 {
		t.Errorf("status=%d want 503", res.StatusCode)
	}
}

func TestParseLimit(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", defaultListLimit},
		{"0", defaultListLimit},
		{"-3", defaultListLimit},
		{"abc", defaultListLimit},
		{"5", 5},
		{"500", maxListLimit},
	}
	for _, tc := range cases {
		if got := parseLimit(tc.in); got != tc.want {
			t.Errorf("parseLimit(%q)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestGetReview_IncludesFiles(t *testing.T) {
	s := newTestStore(t)
	payload, _ := json.Marshal(cachedPayload{
		Title: "with files",
		Files: []gh.File{
			{Path: "scanner.go", Status: "modified", Patch: "@@ -1 +1 @@", Additions: 1, Deletions: 1},
			{Path: "README.md", Status: "added", Patch: "@@ -0 +1 @@\n+new", Additions: 1, Deletions: 0},
		},
		Summary:     "s",
		Risks:       json.RawMessage(`[]`),
		Suggestions: json.RawMessage(`[]`),
	})
	id := store.NewID()
	rec := &store.Record{
		ID: id, Owner: "o", Repo: "r", PRNumber: 1, HeadSHA: "sha",
		Payload: payload, CreatedAt: time.Unix(1000, 0),
	}
	if err := s.Put(context.Background(), rec); err != nil {
		t.Fatalf("seed: %v", err)
	}

	srv := startTestServer(t, Deps{Provider: llm.NewMockProvider(), Store: s})
	res, body := getJSON(t, srv, "/api/reviews/"+id)
	if res.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	var d map[string]any
	if err := json.Unmarshal([]byte(body), &d); err != nil {
		t.Fatalf("decode: %v", err)
	}
	files, _ := d["files"].([]any)
	if len(files) != 2 {
		t.Fatalf("files 应 2 个，得到 %v", files)
	}
	f0, _ := files[0].(map[string]any)
	if f0["path"] != "scanner.go" || f0["status"] != "modified" {
		t.Errorf("files[0] 字段错: %+v", f0)
	}
	if _, leak := f0["full_text"]; leak {
		t.Error("FullText 不应序列化")
	}
}

func TestListReviews_ExcludesFiles(t *testing.T) {
	// list 端不应返 files（每条体积大、列表 50 条爆响应）
	s := newTestStore(t)
	payload, _ := json.Marshal(cachedPayload{
		Title: "with files",
		Files: []gh.File{
			{Path: "scanner.go", Status: "modified", Patch: "@@ -1 +1 @@"},
		},
		Summary:     "s",
		Risks:       json.RawMessage(`[]`),
		Suggestions: json.RawMessage(`[]`),
	})
	rec := &store.Record{
		ID: store.NewID(), Owner: "o", Repo: "r", PRNumber: 1, HeadSHA: "sha",
		Payload: payload, CreatedAt: time.Unix(1000, 0),
	}
	if err := s.Put(context.Background(), rec); err != nil {
		t.Fatalf("seed: %v", err)
	}

	srv := startTestServer(t, Deps{Provider: llm.NewMockProvider(), Store: s})
	res, body := getJSON(t, srv, "/api/reviews")
	if res.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	var list []map[string]any
	_ = json.Unmarshal([]byte(body), &list)
	if len(list) == 0 {
		t.Fatal("应有 1 条")
	}
	if _, hasFiles := list[0]["files"]; hasFiles {
		t.Error("list 端不应包含 files 字段（detail 才有）")
	}
}
