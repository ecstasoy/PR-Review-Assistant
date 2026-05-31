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

func TestFindReviewThreadID_MatchesByDatabaseID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "reviewThreads(last:30)") {
			t.Errorf("query missing reviewThreads: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "data": {"repository": {"pullRequest": {"reviewThreads": {"nodes": [
                {"id":"PRT_a","comments":{"nodes":[{"databaseId":100}]}},
                {"id":"PRT_b","comments":{"nodes":[{"databaseId":555}]}}
            ]}}}}
        }`))
	}))
	defer srv.Close()
	c := &Client{HTTPClient: &http.Client{Transport: rewriteHost{base: srv.URL}}}
	id, err := c.FindReviewThreadID(context.Background(), "tok", "o", "r", 42, 555)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if id != "PRT_b" {
		t.Errorf("thread id = %s want PRT_b", id)
	}
}

func TestFindReviewThreadID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[]}}}}}`))
	}))
	defer srv.Close()
	c := &Client{HTTPClient: &http.Client{Transport: rewriteHost{base: srv.URL}}}
	_, err := c.FindReviewThreadID(context.Background(), "tok", "o", "r", 1, 999)
	if err == nil {
		t.Fatal("expected err when no thread matches")
	}
}

func TestApplyReviewThreadSuggestion_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "applyPullRequestReviewThreadSuggestion") {
			t.Errorf("missing mutation: %s", body)
		}
		var req graphQLRequest
		_ = json.Unmarshal(body, &req)
		if req.Variables["id"] != "PRT_xxx" {
			t.Errorf("vars id = %v", req.Variables["id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"applyPullRequestReviewThreadSuggestion":{
            "pullRequest":{"headRef":{"target":{"oid":"abc123def"}}},
            "pullRequestReviewThread":{"isResolved":true}
        }}}`))
	}))
	defer srv.Close()
	c := &Client{HTTPClient: &http.Client{Transport: rewriteHost{base: srv.URL}}}
	r, err := c.ApplyReviewThreadSuggestion(context.Background(), "tok", "PRT_xxx")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if r.CommitOID != "abc123def" {
		t.Errorf("commit oid = %s", r.CommitOID)
	}
	if !r.IsResolved {
		t.Error("thread should be resolved after apply")
	}
}

func TestApplyReviewThreadSuggestion_GraphQLErrorTreatedAsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// GraphQL 200 + errors body 也要被识别为失败
		_, _ = w.Write([]byte(`{"data":null,"errors":[{"message":"Resource not accessible by integration","type":"FORBIDDEN"}]}`))
	}))
	defer srv.Close()
	c := &Client{HTTPClient: &http.Client{Transport: rewriteHost{base: srv.URL}}}
	_, err := c.ApplyReviewThreadSuggestion(context.Background(), "tok", "PRT_x")
	if err == nil {
		t.Fatal("expected error on graphql errors[]")
	}
	if !strings.Contains(err.Error(), "Resource not accessible") {
		t.Errorf("err missing github msg: %v", err)
	}
}
