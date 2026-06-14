package prompts_test

import (
	"strings"
	"testing"

	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prctx"
	"github.com/ecstasoy/PR-Review-Assistant/backend/internal/prompts"
)

var allTemplates = []string{"summary.tmpl", "risks.tmpl", "suggestions.tmpl"}

func render(t *testing.T, name string, c prctx.Context) string {
	t.Helper()
	tmpl, err := prompts.Parse(name)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, c); err != nil {
		t.Fatalf("execute %s: %v", name, err)
	}
	return sb.String()
}

// 被预算丢弃的文件必须出现在 prompt 里，让模型知道它没看到这些改动。
func TestTemplates_IncludeDroppedFiles(t *testing.T) {
	c := prctx.Context{
		L1Meta:  "仓库: o/r#1",
		L2Files: []prctx.FileContext{{Path: "a.go", Patch: "@@ -1 +1 @@"}},
		BudgetReport: prctx.BudgetReport{
			Dropped: []string{"big/gen.pb.go", "vendor/huge.go"},
		},
	}
	for _, name := range allTemplates {
		out := render(t, name, c)
		if !strings.Contains(out, "big/gen.pb.go") || !strings.Contains(out, "vendor/huge.go") {
			t.Errorf("%s 未列出被丢弃文件:\n%s", name, out)
		}
	}
}

// 无丢弃文件时不应出现「未纳入」段落，避免误导模型。
func TestTemplates_OmitDroppedSectionWhenEmpty(t *testing.T) {
	c := prctx.Context{
		L1Meta:  "仓库: o/r#1",
		L2Files: []prctx.FileContext{{Path: "a.go", Patch: "@@ -1 +1 @@"}},
	}
	for _, name := range allTemplates {
		out := render(t, name, c)
		if strings.Contains(out, "未纳入") {
			t.Errorf("%s 在无丢弃文件时不应出现未纳入段落:\n%s", name, out)
		}
	}
}
