package index

import (
	"context"
	"testing"
)

func newRetrieverWithMock(t *testing.T) *SQLiteRetriever {
	t.Helper()
	r, err := NewSQLiteRetriever(":memory:", NewMockEmbedder())
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func TestSQLiteRetriever_UpsertAndRetrieve(t *testing.T) {
	r := newRetrieverWithMock(t)
	ctx := context.Background()
	err := r.UpsertMany(ctx, "owner/repo", []Chunk{
		{Path: "main.go", Idx: 0, Content: "package main\nfunc main() { fmt.Println(\"hi\") }"},
		{Path: "util.go", Idx: 0, Content: "package util\nfunc Add(a, b int) int { return a + b }"},
		{Path: "README.md", Idx: 0, Content: "# project\nthis is a test project"},
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	// query 跟某 chunk 完全一样 → 它自己 cosine=1 排第一
	hits, err := r.Retrieve(ctx, "owner/repo", "package util\nfunc Add(a, b int) int { return a + b }", 2)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("want top-2, got %d", len(hits))
	}
	if hits[0].File != "util.go" {
		t.Errorf("expected util.go first, got %s (score line %q)", hits[0].File, hits[0].Reason)
	}
}

func TestSQLiteRetriever_ScopeIsolation(t *testing.T) {
	r := newRetrieverWithMock(t)
	ctx := context.Background()
	_ = r.UpsertMany(ctx, "alice/foo", []Chunk{{Path: "x.go", Idx: 0, Content: "shared content"}})
	_ = r.UpsertMany(ctx, "bob/bar", []Chunk{{Path: "y.go", Idx: 0, Content: "shared content"}})

	a, _ := r.Retrieve(ctx, "alice/foo", "shared content", 5)
	b, _ := r.Retrieve(ctx, "bob/bar", "shared content", 5)
	if len(a) != 1 || a[0].File != "x.go" {
		t.Errorf("alice scope leaked: %+v", a)
	}
	if len(b) != 1 || b[0].File != "y.go" {
		t.Errorf("bob scope leaked: %+v", b)
	}
}

func TestSQLiteRetriever_UpsertConflictOverwrites(t *testing.T) {
	r := newRetrieverWithMock(t)
	ctx := context.Background()
	_ = r.UpsertMany(ctx, "s", []Chunk{{Path: "a.go", Idx: 0, Content: "version one"}})
	_ = r.UpsertMany(ctx, "s", []Chunk{{Path: "a.go", Idx: 0, Content: "version two"}})

	hits, _ := r.Retrieve(ctx, "s", "version two", 5)
	if len(hits) != 1 {
		t.Fatalf("want 1 hit (overwrite), got %d", len(hits))
	}
	if hits[0].Snippet != "version two" {
		t.Errorf("overwrite failed; got snippet %q", hits[0].Snippet)
	}
}

func TestSQLiteRetriever_EmptyQuery(t *testing.T) {
	r := newRetrieverWithMock(t)
	hits, err := r.Retrieve(context.Background(), "s", "", 5)
	if err != nil || hits != nil {
		t.Errorf("empty query should return (nil,nil); got=(%v,%v)", hits, err)
	}
}

func TestSQLiteRetriever_KCapping(t *testing.T) {
	r := newRetrieverWithMock(t)
	ctx := context.Background()
	for i := range 8 {
		_ = r.UpsertMany(ctx, "s", []Chunk{
			{Path: "f.go", Idx: i, Content: "chunk content " + string(rune('a'+i))},
		})
	}
	hits, _ := r.Retrieve(ctx, "s", "chunk content x", 3)
	if len(hits) != 3 {
		t.Errorf("want k=3, got %d", len(hits))
	}
}

func TestEncodeDecodeVec_RoundTrip(t *testing.T) {
	in := []float32{0.1, -0.5, 1e-6, 3.14}
	out, err := decodeVec(encodeVec(in))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != len(in) {
		t.Fatalf("len mismatch")
	}
	for i := range in {
		if out[i] != in[i] {
			t.Errorf("mismatch at %d: got %f want %f", i, out[i], in[i])
		}
	}
}

// TestSQLiteRetriever_PRNumberRoundTrip 验证 IndexerChunk.PRNumber 写入 + Retrieve 读回。
// LLM prompt 依赖此字段显示「来自 PR #X」让 agent 能引用其他 PR 的代码。
func TestSQLiteRetriever_PRNumberRoundTrip(t *testing.T) {
	r := newRetrieverWithMock(t)
	ctx := context.Background()
	if err := r.UpsertMany(ctx, "o/r", []Chunk{
		{Path: "a.go", Idx: 0, Content: "from PR 76", PRNumber: 76},
		{Path: "b.go", Idx: 0, Content: "from PR 80", PRNumber: 80},
		{Path: "c.go", Idx: 0, Content: "unknown origin"}, // PRNumber=0 测兼容
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	hits, err := r.Retrieve(ctx, "o/r", "from PR 76", 3)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	byFile := map[string]Reference{}
	for _, h := range hits {
		byFile[h.File] = h
	}
	if byFile["a.go"].PRNumber != 76 {
		t.Errorf("a.go PRNumber = %d, want 76", byFile["a.go"].PRNumber)
	}
	if byFile["b.go"].PRNumber != 80 {
		t.Errorf("b.go PRNumber = %d, want 80", byFile["b.go"].PRNumber)
	}
	if byFile["c.go"].PRNumber != 0 {
		t.Errorf("c.go PRNumber = %d, want 0 (unknown origin)", byFile["c.go"].PRNumber)
	}
}

func TestNewSQLiteRetriever_RejectsNilEmbedder(t *testing.T) {
	_, err := NewSQLiteRetriever(":memory:", nil)
	if err == nil {
		t.Error("expected err on nil embedder")
	}
}
