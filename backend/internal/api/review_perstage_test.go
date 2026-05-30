package api

import (
	"context"
	"strings"
	"testing"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/index"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
)

// recordingBuilder 透传 base Build，但记录 BuildWith 收到的 RAGQuery，便于断言 per-stage 真传了不同 query
type recordingBuilder struct {
	queries []string
}

func (r *recordingBuilder) Build(ctx context.Context, pr gh.PullRequest) (prctx.Context, error) {
	return r.BuildWith(ctx, pr, prctx.BuildOptions{})
}

func (r *recordingBuilder) BuildWith(_ context.Context, _ gh.PullRequest, opts prctx.BuildOptions) (prctx.Context, error) {
	r.queries = append(r.queries, opts.RAGQuery)
	return prctx.Context{L1Meta: "stub"}, nil
}

func TestStageRAGQueryFor(t *testing.T) {
	pr := gh.PullRequest{Files: []gh.File{{Path: "a.go"}, {Path: "b.go"}}}
	cases := []struct {
		stage   string
		wantSub []string // query 应含这些子串
	}{
		{"summary", []string{}},
		{"risks", []string{"bug", "race", "a.go", "b.go"}},
		{"suggestions", []string{"重构", "a.go", "b.go"}},
		{"unknown", []string{}},
	}
	for _, tc := range cases {
		got := stageRAGQueryFor(tc.stage, pr)
		if len(tc.wantSub) == 0 {
			if got != "" {
				t.Errorf("stage=%s: want empty query, got %q", tc.stage, got)
			}
			continue
		}
		for _, sub := range tc.wantSub {
			if !strings.Contains(got, sub) {
				t.Errorf("stage=%s: query missing %q\ngot: %s", tc.stage, sub, got)
			}
		}
	}
}

func TestBuildPerStageContexts_CallsBuildWithQuery(t *testing.T) {
	rb := &recordingBuilder{}
	pr := gh.PullRequest{Files: []gh.File{{Path: "main.go"}}}
	base := prctx.Context{L1Meta: "base"}

	ctxs := buildPerStageContexts(context.Background(), rb, pr, base)

	// summary 必须复用 base 不再调 BuildWith；risks/suggestions 必须各调一次（带 query）
	if got := len(rb.queries); got != 2 {
		t.Fatalf("expected 2 BuildWith calls (risks + suggestions), got %d (queries=%+v)", got, rb.queries)
	}
	for _, q := range rb.queries {
		if q == "" {
			t.Errorf("per-stage query should be non-empty; got: %v", rb.queries)
		}
	}
	if ctxs["summary"].L1Meta != base.L1Meta {
		t.Errorf("summary should reuse base; got %+v", ctxs["summary"])
	}
}

// failingBuilder Build/BuildWith 都报错；验证 buildPerStageContexts fallback 到 base
type failingBuilder struct{}

func (failingBuilder) Build(_ context.Context, _ gh.PullRequest) (prctx.Context, error) {
	return prctx.Context{}, errBuilder
}
func (failingBuilder) BuildWith(_ context.Context, _ gh.PullRequest, _ prctx.BuildOptions) (prctx.Context, error) {
	return prctx.Context{}, errBuilder
}

var errBuilder = stubErr("builder failure")

type stubErr string

func (e stubErr) Error() string { return string(e) }

func TestBuildPerStageContexts_FallsBackOnError(t *testing.T) {
	base := prctx.Context{L1Meta: "the-base", L4References: []index.Reference{{File: "x.go"}}}
	pr := gh.PullRequest{Files: []gh.File{{Path: "a.go"}}}
	ctxs := buildPerStageContexts(context.Background(), failingBuilder{}, pr, base)
	for _, name := range []string{"summary", "risks", "suggestions"} {
		if ctxs[name].L1Meta != "the-base" {
			t.Errorf("stage=%s should fallback to base on err; got %+v", name, ctxs[name])
		}
	}
}
