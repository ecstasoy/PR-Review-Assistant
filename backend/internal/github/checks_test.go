package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gh "github.com/google/go-github/v66/github"
)

func ghClient(t *testing.T, h http.Handler) (*gh.Client, func()) {
	t.Helper()
	srv := httptest.NewServer(h)
	c := gh.NewClient(nil)
	base, _ := url.Parse(srv.URL + "/")
	c.BaseURL = base
	return c, srv.Close
}

// checksHandler 构造 GitHub check-runs list 响应；conclusions 中传 "" 表示 status != completed。
func checksHandler(names, conclusions []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runs := make([]map[string]any, 0, len(names))
		for i, n := range names {
			run := map[string]any{
				"name":         n,
				"started_at":   "2026-05-28T10:00:00Z",
				"completed_at": "2026-05-28T10:00:05Z",
			}
			if conclusions[i] == "" {
				run["status"] = "in_progress"
			} else {
				run["status"] = "completed"
				run["conclusion"] = conclusions[i]
			}
			runs = append(runs, run)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_count": len(runs),
			"check_runs":  runs,
		})
	}
}

func TestFetchChecks_AllPassing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/commits/sha/check-runs",
		checksHandler([]string{"build", "test", "lint"},
			[]string{"success", "success", "success"}))
	c, cleanup := ghClient(t, mux)
	defer cleanup()

	ci, checks, err := fetchChecks(context.Background(), c, "o", "r", "sha")
	if err != nil {
		t.Fatalf("fetchChecks: %v", err)
	}
	if ci != CIStatusPassing {
		t.Errorf("ci=%q want passing", ci)
	}
	if len(checks) != 3 {
		t.Fatalf("checks len=%d want 3", len(checks))
	}
	for _, c := range checks {
		if c.Status != CIStatusPassing {
			t.Errorf("check %s status=%q want passing", c.Name, c.Status)
		}
		if c.DurationMS != 5000 {
			t.Errorf("check %s duration=%d want 5000ms", c.Name, c.DurationMS)
		}
	}
}

func TestFetchChecks_AnyFailing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/commits/sha/check-runs",
		checksHandler([]string{"build", "test", "lint"},
			[]string{"success", "failure", "success"}))
	c, cleanup := ghClient(t, mux)
	defer cleanup()

	ci, _, _ := fetchChecks(context.Background(), c, "o", "r", "sha")
	if ci != CIStatusFailing {
		t.Errorf("有一个 failure 时 ci 应为 failing，得到 %q", ci)
	}
}

func TestFetchChecks_AnyPendingNoneFailing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/commits/sha/check-runs",
		checksHandler([]string{"build", "coverage"},
			[]string{"success", ""}))
	c, cleanup := ghClient(t, mux)
	defer cleanup()

	ci, _, _ := fetchChecks(context.Background(), c, "o", "r", "sha")
	if ci != CIStatusPending {
		t.Errorf("有一个未完成且无失败时 ci 应为 pending，得到 %q", ci)
	}
}

func TestFetchChecks_FailingTrumpsPending(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/commits/sha/check-runs",
		checksHandler([]string{"build", "test"},
			[]string{"failure", ""}))
	c, cleanup := ghClient(t, mux)
	defer cleanup()

	ci, _, _ := fetchChecks(context.Background(), c, "o", "r", "sha")
	if ci != CIStatusFailing {
		t.Errorf("failing 应优先 pending，得到 %q", ci)
	}
}

func TestFetchChecks_EmptyReturnsPending(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/commits/sha/check-runs",
		checksHandler(nil, nil)) // 空数组
	c, cleanup := ghClient(t, mux)
	defer cleanup()

	ci, checks, err := fetchChecks(context.Background(), c, "o", "r", "sha")
	if err != nil {
		t.Fatalf("fetchChecks: %v", err)
	}
	if ci != CIStatusPending {
		t.Errorf("空 checks 应视为 pending，得到 %q", ci)
	}
	if checks == nil {
		t.Error("checks 应是空 slice 而非 nil")
	}
}

func TestFetchChecks_NeutralAndSkippedArePending(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/commits/sha/check-runs",
		checksHandler([]string{"build", "deploy", "audit"},
			[]string{"success", "skipped", "neutral"}))
	c, cleanup := ghClient(t, mux)
	defer cleanup()

	ci, checks, _ := fetchChecks(context.Background(), c, "o", "r", "sha")
	if ci != CIStatusPending {
		t.Errorf("含 neutral / skipped 时整体应为 pending，得到 %q", ci)
	}
	for _, c := range checks {
		if c.Name == "build" && c.Status != CIStatusPassing {
			t.Errorf("build 应 passing")
		}
		if (c.Name == "deploy" || c.Name == "audit") && c.Status != CIStatusPending {
			t.Errorf("%s 应 pending，得到 %s", c.Name, c.Status)
		}
	}
}

func TestFetchChecks_500Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/commits/sha/check-runs", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
	})
	c, cleanup := ghClient(t, mux)
	defer cleanup()

	_, _, err := fetchChecks(context.Background(), c, "o", "r", "sha")
	if err == nil {
		t.Fatal("500 应返 error")
	}
}

// 端到端：RealFetcher.Fetch 应把 CI / Checks 字段填到 PullRequest
func TestRealFetcher_Fetch_AttachesCI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"title": "test",
			"head":  map[string]any{"sha": "abc"},
		})
	})
	mux.HandleFunc("/repos/o/r/pulls/1/files", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	})
	mux.HandleFunc("/repos/o/r/commits/abc/check-runs",
		checksHandler([]string{"build"}, []string{"success"}))

	srv := httptest.NewServer(mux)
	defer srv.Close()
	f := NewRealFetcher("")
	base, _ := url.Parse(srv.URL + "/")
	f.client.BaseURL = base

	got, err := f.Fetch(context.Background(), "https://github.com/o/r/pull/1")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got.CI != CIStatusPassing {
		t.Errorf("CI=%q want passing", got.CI)
	}
	if len(got.Checks) != 1 || got.Checks[0].Name != "build" {
		t.Errorf("Checks 错: %+v", got.Checks)
	}
}

// 端到端：CI 抓取失败不阻塞主流程，CI 字段保持空字符串
func TestRealFetcher_Fetch_CIErrorDegrades(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"title": "test",
			"head":  map[string]any{"sha": "abc"},
		})
	})
	mux.HandleFunc("/repos/o/r/pulls/1/files", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	})
	mux.HandleFunc("/repos/o/r/commits/abc/check-runs", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"forbidden"}`, http.StatusForbidden)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	f := NewRealFetcher("")
	base, _ := url.Parse(srv.URL + "/")
	f.client.BaseURL = base

	got, err := f.Fetch(context.Background(), "https://github.com/o/r/pull/1")
	if err != nil {
		t.Fatalf("Fetch 不应因 CI 失败而报错: %v", err)
	}
	if got.CI != "" {
		t.Errorf("CI 抓取失败时 CI 应留空（区分 unknown 与 pending），得到 %q", got.CI)
	}
	if len(got.Checks) != 0 {
		t.Errorf("失败时 Checks 应空，得到 %d 条", len(got.Checks))
	}

	// 标题等其它字段仍应正常拉到
	if !strings.Contains(got.Title, "test") {
		t.Errorf("Title 错: %q", got.Title)
	}
}
