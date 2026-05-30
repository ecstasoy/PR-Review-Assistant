package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	gh "github.com/ecstasoy/PR-Review-Assistant/backend/internal/github"
)

func sampleFiles() []gh.File {
	return []gh.File{
		{Path: "main.go", Status: "modified", Patch: "@@ -1,2 +1,3 @@\n package main\n+// TODO fix race\n var x = 1", Additions: 1, Deletions: 0},
		{Path: "util/helper.go", Status: "added", Patch: "@@ -0,0 +1,5 @@\n+package util\n+\n+func Helper() string {\n+\treturn \"ok\"\n+}", Additions: 5, Deletions: 0},
		{Path: "README.md", Status: "modified", Patch: "@@ -1 +1,2 @@\n # PR-Review\n+## Quick start", Additions: 1, Deletions: 0},
	}
}

func TestReadFileTool_Found(t *testing.T) {
	tool := NewReadFileTool(sampleFiles())
	out, err := tool.Run(context.Background(), json.RawMessage(`{"file":"main.go"}`))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "patch:") || !strings.Contains(out, "TODO fix race") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestReadFileTool_SandboxRejects(t *testing.T) {
	tool := NewReadFileTool(sampleFiles())
	_, err := tool.Run(context.Background(), json.RawMessage(`{"file":"/etc/passwd"}`))
	if err == nil || !strings.Contains(err.Error(), "沙盒") {
		t.Errorf("want sandbox rejection, got err=%v", err)
	}
}

func TestReadFileTool_MissingArg(t *testing.T) {
	tool := NewReadFileTool(sampleFiles())
	_, err := tool.Run(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "file 不能为空") {
		t.Errorf("want missing-file error, got %v", err)
	}
}

func TestListDirTool_Filtered(t *testing.T) {
	tool := NewListDirTool(sampleFiles())
	out, err := tool.Run(context.Background(), json.RawMessage(`{"prefix":"util/"}`))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "util/helper.go") {
		t.Errorf("util/helper.go missing in output: %s", out)
	}
	if strings.Contains(out, "main.go") {
		t.Errorf("main.go should be filtered out, but appears in: %s", out)
	}
}

func TestListDirTool_NoPrefix_ListsAll(t *testing.T) {
	tool := NewListDirTool(sampleFiles())
	out, _ := tool.Run(context.Background(), json.RawMessage(`{}`))
	for _, path := range []string{"main.go", "util/helper.go", "README.md"} {
		if !strings.Contains(out, path) {
			t.Errorf("path %s missing: %s", path, out)
		}
	}
}

func TestGrepTool_LiteralMatch(t *testing.T) {
	tool := NewGrepTool(sampleFiles())
	out, err := tool.Run(context.Background(), json.RawMessage(`{"pattern":"TODO"}`))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "main.go:") || !strings.Contains(out, "TODO fix race") {
		t.Errorf("unexpected: %s", out)
	}
}

func TestGrepTool_Regex(t *testing.T) {
	tool := NewGrepTool(sampleFiles())
	out, err := tool.Run(context.Background(), json.RawMessage(`{"pattern":"^\\+.*func","regex":true}`))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(out, "func Helper") {
		t.Errorf("regex hit expected, got: %s", out)
	}
}

func TestGrepTool_NoMatch(t *testing.T) {
	tool := NewGrepTool(sampleFiles())
	out, _ := tool.Run(context.Background(), json.RawMessage(`{"pattern":"impossible-zzz-xyz"}`))
	if !strings.Contains(out, "无命中") {
		t.Errorf("want 无命中 message, got: %s", out)
	}
}

func TestRegisterDefaults_AllThree(t *testing.T) {
	r := NewRegistry()
	RegisterDefaults(r, sampleFiles())
	for _, name := range []string{"read_file", "list_dir", "grep_patches"} {
		if _, ok := r.Lookup(name); !ok {
			t.Errorf("default tool %s missing after RegisterDefaults", name)
		}
	}
}
