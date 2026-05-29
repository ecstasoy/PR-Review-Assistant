package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gh "github.com/google/go-github/v66/github"
)

// stubClient 给 fetchConventions 跑用的 go-github client，BaseURL 指向 mux
func stubClient(t *testing.T, handler http.Handler) (*gh.Client, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := gh.NewClient(nil)
	base, _ := url.Parse(srv.URL + "/")
	c.BaseURL = base
	return c, srv.Close
}

// contentsHandler 返回单文件 contents API 响应（base64 编码 body）
func contentsHandler(name, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":     "file",
			"name":     name,
			"path":     name,
			"encoding": "base64",
			"content":  base64.StdEncoding.EncodeToString([]byte(body)),
			"size":     len(body),
		})
	}
}

func TestFetchConventions_AllPresent(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/contents/README.md", contentsHandler("README.md", "# Repo\nUse Go 1.22+"))
	mux.HandleFunc("/repos/o/r/contents/CONTRIBUTING.md", contentsHandler("CONTRIBUTING.md", "Open PR before commit"))
	mux.HandleFunc("/repos/o/r/contents/CLAUDE.md", contentsHandler("CLAUDE.md", "errors.Is for sentinels"))

	c, cleanup := stubClient(t, mux)
	defer cleanup()

	conv, err := fetchConventions(context.Background(), c, "o", "r", "abc123")
	if err != nil {
		t.Fatalf("fetchConventions: %v", err)
	}
	if !strings.Contains(conv.Readme, "Use Go 1.22+") {
		t.Errorf("Readme 缺内容: %q", conv.Readme)
	}
	if !strings.Contains(conv.Contributing, "Open PR") {
		t.Errorf("Contributing 缺内容: %q", conv.Contributing)
	}
	if !strings.Contains(conv.AgentDocs, "errors.Is") {
		t.Errorf("AgentDocs 缺内容: %q", conv.AgentDocs)
	}
}

func TestFetchConventions_OnlyReadme(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/contents/README.md", contentsHandler("README.md", "# Repo"))
	// 其他路径 mux 默认 404

	c, cleanup := stubClient(t, mux)
	defer cleanup()

	conv, err := fetchConventions(context.Background(), c, "o", "r", "")
	if err != nil {
		t.Fatalf("fetchConventions: %v", err)
	}
	if conv.Readme == "" {
		t.Error("Readme 应有内容")
	}
	if conv.Contributing != "" {
		t.Errorf("Contributing 应空: %q", conv.Contributing)
	}
	if conv.AgentDocs != "" {
		t.Errorf("AgentDocs 应空: %q", conv.AgentDocs)
	}
}

func TestFetchConventions_PreferCLAUDEOverAGENTS(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/contents/CLAUDE.md", contentsHandler("CLAUDE.md", "claude-md content"))
	mux.HandleFunc("/repos/o/r/contents/AGENTS.md", contentsHandler("AGENTS.md", "agents-md content"))

	c, cleanup := stubClient(t, mux)
	defer cleanup()

	conv, err := fetchConventions(context.Background(), c, "o", "r", "")
	if err != nil {
		t.Fatalf("fetchConventions: %v", err)
	}
	if !strings.Contains(conv.AgentDocs, "claude-md") {
		t.Errorf("应优先取 CLAUDE.md，得到 %q", conv.AgentDocs)
	}
}

func TestFetchConventions_FallbackToAGENTS(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/contents/AGENTS.md", contentsHandler("AGENTS.md", "agents-md content"))
	// 不挂 CLAUDE.md → 404 fallback

	c, cleanup := stubClient(t, mux)
	defer cleanup()

	conv, err := fetchConventions(context.Background(), c, "o", "r", "")
	if err != nil {
		t.Fatalf("fetchConventions: %v", err)
	}
	if !strings.Contains(conv.AgentDocs, "agents-md") {
		t.Errorf("缺 CLAUDE.md 时应回落 AGENTS.md，得到 %q", conv.AgentDocs)
	}
}

func TestFetchConventions_AllAbsent_NoError(t *testing.T) {
	mux := http.NewServeMux() // 所有路径都 404

	c, cleanup := stubClient(t, mux)
	defer cleanup()

	conv, err := fetchConventions(context.Background(), c, "o", "r", "")
	if err != nil {
		t.Fatalf("全 404 不应报错: %v", err)
	}
	if conv != (Conventions{}) {
		t.Errorf("应返回空 Conventions，得到 %+v", conv)
	}
}

func TestFetchConventions_TooLargeTruncated(t *testing.T) {
	huge := strings.Repeat("X", maxConventionFileSize+5000)
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/contents/README.md", contentsHandler("README.md", huge))

	c, cleanup := stubClient(t, mux)
	defer cleanup()

	conv, err := fetchConventions(context.Background(), c, "o", "r", "")
	if err != nil {
		t.Fatalf("fetchConventions: %v", err)
	}
	if !strings.HasSuffix(conv.Readme, "truncated]") {
		t.Errorf("超大文件应截断带标记: 末尾=%q", conv.Readme[max(0, len(conv.Readme)-30):])
	}
	if len(conv.Readme) > maxConventionFileSize+50 {
		t.Errorf("截断后长度应 ≈ maxConventionFileSize；得到 %d", len(conv.Readme))
	}
}

func TestFetchConventions_NonNotFoundError_Propagates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/contents/README.md", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"server boom"}`, http.StatusInternalServerError)
	})

	c, cleanup := stubClient(t, mux)
	defer cleanup()

	_, err := fetchConventions(context.Background(), c, "o", "r", "")
	if err == nil {
		t.Fatal("500 应传递为 error")
	}
}
