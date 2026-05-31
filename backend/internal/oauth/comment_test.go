package oauth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostPRComment_BuildsCorrectPayloadAndParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/repos/octo/repo/pulls/42/comments") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		var p map[string]any
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &p)
		if p["body"] != "hello\n```suggestion\nnew\n```" {
			t.Errorf("body = %v", p["body"])
		}
		if p["commit_id"] != "sha123" {
			t.Errorf("commit_id = %v", p["commit_id"])
		}
		if p["path"] != "main.go" {
			t.Errorf("path = %v", p["path"])
		}
		if int(p["line"].(float64)) != 7 {
			t.Errorf("line = %v", p["line"])
		}
		if p["side"] != "RIGHT" {
			t.Errorf("side = %v", p["side"])
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":555,"node_id":"PRC_xxx","html_url":"https://github.com/octo/repo/pull/42#discussion_r555","body":"hello","path":"main.go","line":7}`))
	}))
	defer srv.Close()
	c := &Client{HTTPClient: &http.Client{Transport: rewriteHost{base: srv.URL}}}
	cm, err := c.PostPRComment(context.Background(), "tok", "octo", "repo", 42,
		"hello\n```suggestion\nnew\n```", "sha123", "main.go", 7)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if cm.ID != 555 || cm.NodeID != "PRC_xxx" {
		t.Errorf("comment fields = %+v", cm)
	}
	if cm.HTMLUrl == "" {
		t.Errorf("missing html_url")
	}
}

func TestPostPRComment_GitHubError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Pull request review thread line must be part of the diff"}`))
	}))
	defer srv.Close()
	c := &Client{HTTPClient: &http.Client{Transport: rewriteHost{base: srv.URL}}}
	_, err := c.PostPRComment(context.Background(), "tok", "o", "r", 1, "body", "sha", "f", 1)
	if err == nil {
		t.Fatal("expected error on 422")
	}
	if !strings.Contains(err.Error(), "Pull request review thread") {
		t.Errorf("err missing github msg: %v", err)
	}
}
