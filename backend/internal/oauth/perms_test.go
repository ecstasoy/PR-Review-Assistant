package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRepoPermission_CanComment_Matrix(t *testing.T) {
	cases := []struct {
		perm RepoPermission
		want bool
	}{
		{PermAdmin, true},
		{PermMaintain, true},
		{PermWrite, true},
		{PermTriage, true},
		{PermRead, false},
		{PermNone, false},
		{RepoPermission("unknown"), false},
	}
	for _, c := range cases {
		if got := c.perm.CanComment(); got != c.want {
			t.Errorf("perm=%s CanComment=%v want %v", c.perm, got, c.want)
		}
	}
}

func TestRepoPermission_CanCommit_TriageCannotCommit(t *testing.T) {
	cases := []struct {
		perm RepoPermission
		want bool
	}{
		{PermAdmin, true},
		{PermMaintain, true},
		{PermWrite, true},
		{PermTriage, false}, // 关键：triage 可评论但不能 push
		{PermRead, false},
		{PermNone, false},
	}
	for _, c := range cases {
		if got := c.perm.CanCommit(); got != c.want {
			t.Errorf("perm=%s CanCommit=%v want %v", c.perm, got, c.want)
		}
	}
}

func TestGetRepoPermission_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/repos/octo/repo/collaborators/alice/permission") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer ghu_test" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"permission":"write","user":{"login":"alice"}}`))
	}))
	defer srv.Close()
	c := &Client{HTTPClient: &http.Client{Transport: rewriteHost{base: srv.URL}}}
	perm, err := c.GetRepoPermission(context.Background(), "ghu_test", "octo", "repo", "alice")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if perm != PermWrite {
		t.Errorf("perm = %s, want write", perm)
	}
}

func TestGetRepoPermission_404ReturnsNoneWithoutError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()
	c := &Client{HTTPClient: &http.Client{Transport: rewriteHost{base: srv.URL}}}
	perm, err := c.GetRepoPermission(context.Background(), "tok", "o", "r", "u")
	if err != nil {
		t.Fatalf("404 should not be error; got %v", err)
	}
	if perm != PermNone {
		t.Errorf("404 perm = %s, want none", perm)
	}
}

func TestGetRepoPermission_403ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()
	c := &Client{HTTPClient: &http.Client{Transport: rewriteHost{base: srv.URL}}}
	_, err := c.GetRepoPermission(context.Background(), "bad", "o", "r", "u")
	if err == nil {
		t.Fatal("expected err on 403")
	}
}
