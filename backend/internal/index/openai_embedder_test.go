package index

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// stubEmbeddingsServer httptest 模拟 /v1/embeddings，让 handler 拿 request body 写 SSE/JSON 响应
func stubEmbeddingsServer(t *testing.T, handler func(t *testing.T, w http.ResponseWriter, body []byte)) *OpenAIEmbedder {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing Authorization header")
		}
		body, _ := io.ReadAll(r.Body)
		handler(t, w, body)
	}))
	t.Cleanup(srv.Close)
	return NewOpenAIEmbedder(srv.URL, "test-key", "test-model")
}

func TestOpenAIEmbedder_Embed_Success(t *testing.T) {
	e := stubEmbeddingsServer(t, func(t *testing.T, w http.ResponseWriter, body []byte) {
		var req openAIEmbeddingsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Model != "test-model" {
			t.Errorf("model=%q want test-model", req.Model)
		}
		if len(req.Input) != 2 {
			t.Errorf("want 2 inputs, got %d", len(req.Input))
		}
		// 模拟返回；OpenAI data 顺序不保证，故乱序返回测 index 正确归位
		fmt.Fprint(w, `{"data":[
			{"index":1, "embedding":[0.4, 0.5, 0.6]},
			{"index":0, "embedding":[0.1, 0.2, 0.3]}
		]}`)
	})

	got, err := e.Embed(context.Background(), []string{"hello", "world"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 vecs, got %d", len(got))
	}
	if got[0][0] != 0.1 || got[1][0] != 0.4 {
		t.Errorf("index reorder broken: got=%v", got)
	}
}

func TestOpenAIEmbedder_Embed_Empty(t *testing.T) {
	e := NewOpenAIEmbedder("http://invalid", "key", "")
	got, err := e.Embed(context.Background(), nil)
	if err != nil || got != nil {
		t.Errorf("empty input should return (nil,nil); got=(%v,%v)", got, err)
	}
}

func TestOpenAIEmbedder_Embed_Non200(t *testing.T) {
	e := stubEmbeddingsServer(t, func(_ *testing.T, w http.ResponseWriter, _ []byte) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid key"}`))
	})
	_, err := e.Embed(context.Background(), []string{"x"})
	if err == nil {
		t.Error("expected err on 401")
	}
}

func TestOpenAIEmbedder_DefaultModelWhenEmpty(t *testing.T) {
	e := NewOpenAIEmbedder("http://x", "k", "")
	if e.Model != "text-embedding-3-small" {
		t.Errorf("default model wrong: %q", e.Model)
	}
}

func TestMockEmbedder_Deterministic(t *testing.T) {
	m := NewMockEmbedder()
	a, _ := m.Embed(context.Background(), []string{"foo"})
	b, _ := m.Embed(context.Background(), []string{"foo"})
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("want 1 vec each")
	}
	if a[0][0] != b[0][0] || a[0][1535] != b[0][1535] {
		t.Errorf("mock embedder not deterministic")
	}
}

func TestMockEmbedder_DifferentTextsDiffer(t *testing.T) {
	m := NewMockEmbedder()
	out, _ := m.Embed(context.Background(), []string{"foo", "bar"})
	if out[0][0] == out[1][0] && out[0][1] == out[1][1] {
		t.Errorf("different inputs should differ in some dim")
	}
}

func TestMockEmbedder_NormalizedL2(t *testing.T) {
	m := NewMockEmbedder()
	out, _ := m.Embed(context.Background(), []string{"hello"})
	var sum float64
	for _, v := range out[0] {
		sum += float64(v) * float64(v)
	}
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("vec should be L2-normalized; sum_sq=%f", sum)
	}
}

// 编译期断言两实现都满足 Embedder
var (
	_ Embedder = (*OpenAIEmbedder)(nil)
	_ Embedder = (*MockEmbedder)(nil)
)
