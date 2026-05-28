package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// stubServer 起一个 httptest server，go-github 客户端 BaseURL 指向这里
func stubServer(t *testing.T, handler http.Handler) (*RealFetcher, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	f := NewRealFetcher("")
	base, _ := url.Parse(srv.URL + "/")
	f.client.BaseURL = base
	return f, srv.Close
}

func TestRealFetcher_Fetch_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/golang/go/pulls/42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"title": "fix panic in scanner",
			"body":  "fixes #999",
			"head":  map[string]any{"sha": "deadbeef0000"},
		})
	})
	mux.HandleFunc("/repos/golang/go/pulls/42/files", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"filename":  "src/main.go",
				"status":    "modified",
				"patch":     "@@ -1,1 +1,1 @@\n-old\n+new",
				"additions": 1,
				"deletions": 1,
			},
			{
				"filename":  "README.md",
				"status":    "modified",
				"patch":     "@@ -1 +1 @@\n-A\n+B",
				"additions": 1,
				"deletions": 1,
			},
		})
	})

	f, cleanup := stubServer(t, mux)
	defer cleanup()

	got, err := f.Fetch(context.Background(), "https://github.com/golang/go/pull/42")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if got.Owner != "golang" || got.Repo != "go" || got.Number != 42 {
		t.Errorf("PR 元信息错: %+v", got)
	}
	if got.HeadSHA != "deadbeef0000" {
		t.Errorf("HeadSHA = %q，期望 deadbeef0000", got.HeadSHA)
	}
	if got.Title != "fix panic in scanner" {
		t.Errorf("Title = %q", got.Title)
	}
	if got.Body != "fixes #999" {
		t.Errorf("Body = %q", got.Body)
	}
	if len(got.Files) != 2 {
		t.Fatalf("Files 数量 = %d，期望 2", len(got.Files))
	}
	if got.Files[0].Path != "src/main.go" || got.Files[0].Status != "modified" {
		t.Errorf("Files[0] 错: %+v", got.Files[0])
	}
	if got.Files[0].Additions != 1 || got.Files[0].Deletions != 1 {
		t.Errorf("Files[0] 加减行错: +%d -%d", got.Files[0].Additions, got.Files[0].Deletions)
	}
	if got.Files[1].Path != "README.md" {
		t.Errorf("Files[1] 错: %+v", got.Files[1])
	}
}

func TestRealFetcher_Fetch_InvalidURL(t *testing.T) {
	f := NewRealFetcher("")
	_, err := f.Fetch(context.Background(), "https://gitlab.com/foo/bar/pull/1")
	if !errors.Is(err, ErrInvalidPRURL) {
		t.Errorf("期望 ErrInvalidPRURL，得到 %v", err)
	}
}

func TestRealFetcher_Fetch_PRNotFound(t *testing.T) {
	srv := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})
	f, cleanup := stubServer(t, srv)
	defer cleanup()

	_, err := f.Fetch(context.Background(), "https://github.com/owner/repo/pull/1")
	if err == nil {
		t.Fatal("期望 404 错误，但 Fetch 没报错")
	}
}

func TestRealFetcher_Fetch_EmptyFiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"title": "empty PR",
			"head":  map[string]any{"sha": "0"},
		})
	})
	mux.HandleFunc("/repos/o/r/pulls/1/files", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	})

	f, cleanup := stubServer(t, mux)
	defer cleanup()

	got, err := f.Fetch(context.Background(), "https://github.com/o/r/pull/1")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(got.Files) != 0 {
		t.Errorf("期望空 Files，得到 %d 条", len(got.Files))
	}
}
