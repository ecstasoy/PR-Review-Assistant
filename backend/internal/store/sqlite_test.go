package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func sampleRecord(headSHA string) *Record {
	return &Record{
		ID:       NewID(),
		Owner:    "golang",
		Repo:     "go",
		PRNumber: 42,
		HeadSHA:  headSHA,
		Payload:  json.RawMessage(`{"summary":"ok","risks":[],"suggestions":[]}`),
	}
}

func TestSQLiteStore_PutGet_RoundTrip(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	in := sampleRecord("sha-A")
	if err := s.Put(ctx, in); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := s.Get(ctx, "golang", "go", 42, "sha-A")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get 未命中刚写入的记录")
	}
	if got.ID != in.ID {
		t.Errorf("ID 不一致: got=%s, want=%s", got.ID, in.ID)
	}
	if string(got.Payload) != string(in.Payload) {
		t.Errorf("Payload 不一致: got=%s", got.Payload)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt 应被自动填充")
	}
}

func TestSQLiteStore_Get_MissReturnsNilNilNotError(t *testing.T) {
	s := newTestStore(t)

	got, err := s.Get(context.Background(), "x", "y", 1, "nope")
	if err != nil {
		t.Fatalf("Get 未命中不应报错，得到 %v", err)
	}
	if got != nil {
		t.Errorf("Get 未命中应返 nil，得到 %+v", got)
	}
}

func TestSQLiteStore_Put_SameSHAPreservesID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	in1 := sampleRecord("sha-X")
	if err := s.Put(ctx, in1); err != nil {
		t.Fatalf("Put 1: %v", err)
	}

	// 同 (owner, repo, pr, sha) 二次写入：ID 复用，payload + created_at 刷新
	in2 := &Record{
		ID:       NewID(), // 新生成的 ID 应被 ON CONFLICT 忽略
		Owner:    in1.Owner, Repo: in1.Repo, PRNumber: in1.PRNumber, HeadSHA: in1.HeadSHA,
		Payload: json.RawMessage(`{"summary":"updated"}`),
	}
	if err := s.Put(ctx, in2); err != nil {
		t.Fatalf("Put 2: %v", err)
	}

	got, err := s.Get(ctx, in1.Owner, in1.Repo, in1.PRNumber, in1.HeadSHA)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != in1.ID {
		t.Errorf("ID 应保持首次写入的 %s，得到 %s", in1.ID, got.ID)
	}
	if string(got.Payload) != string(in2.Payload) {
		t.Errorf("Payload 应被刷新为 in2，得到 %s", got.Payload)
	}
}

func TestSQLiteStore_Put_EmptyIDRejected(t *testing.T) {
	s := newTestStore(t)
	r := sampleRecord("sha")
	r.ID = ""
	if err := s.Put(context.Background(), r); err == nil {
		t.Error("空 ID 应被拒绝")
	}
}

func TestSQLiteStore_List_OrdersByCreatedAtDesc(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// 用显式 CreatedAt 避免 ULID 时间精度相同导致的不稳定
	older := sampleRecord("sha-old")
	older.CreatedAt = time.Unix(1000, 0)
	if err := s.Put(ctx, older); err != nil {
		t.Fatalf("Put older: %v", err)
	}

	newer := sampleRecord("sha-new")
	newer.PRNumber = 43 // 避开 UNIQUE
	newer.CreatedAt = time.Unix(2000, 0)
	if err := s.Put(ctx, newer); err != nil {
		t.Fatalf("Put newer: %v", err)
	}

	got, err := s.List(ctx, nil, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List 应返 2 条，得到 %d", len(got))
	}
	if got[0].HeadSHA != "sha-new" {
		t.Errorf("最新一条应排首位，得到 %s", got[0].HeadSHA)
	}
}

func TestSQLiteStore_List_FiltersByUserID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	alice := "alice"
	pubRec := sampleRecord("sha-pub")
	if err := s.Put(ctx, pubRec); err != nil {
		t.Fatalf("Put pub: %v", err)
	}
	userRec := sampleRecord("sha-user")
	userRec.PRNumber = 43
	userRec.UserID = &alice
	if err := s.Put(ctx, userRec); err != nil {
		t.Fatalf("Put user: %v", err)
	}

	pubList, _ := s.List(ctx, nil, 10)
	if len(pubList) != 1 || pubList[0].HeadSHA != "sha-pub" {
		t.Errorf("user_id=nil 应只返公共记录，得到 %+v", pubList)
	}
	aliceList, _ := s.List(ctx, &alice, 10)
	if len(aliceList) != 1 || aliceList[0].HeadSHA != "sha-user" {
		t.Errorf("user_id=alice 应只返 alice 的记录，得到 %+v", aliceList)
	}
}

func TestSQLiteStore_List_LimitDefaultsTo50(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	for i := range 60 {
		r := sampleRecord("sha")
		r.PRNumber = i + 1
		if err := s.Put(ctx, r); err != nil {
			t.Fatalf("Put %d: %v", i, err)
		}
	}
	got, _ := s.List(ctx, nil, 0)
	if len(got) != 50 {
		t.Errorf("limit=0 应回落 50，得到 %d", len(got))
	}
}

func TestSQLiteStore_GetByID(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	in := sampleRecord("sha")
	if err := s.Put(ctx, in); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := s.GetByID(ctx, in.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil || got.ID != in.ID {
		t.Errorf("GetByID 命中错: %+v", got)
	}

	miss, err := s.GetByID(ctx, "no-such-id")
	if err != nil {
		t.Fatalf("GetByID miss 不应报错: %v", err)
	}
	if miss != nil {
		t.Errorf("GetByID miss 应返 nil")
	}
}

func TestNewID_UniqueAndSortable(t *testing.T) {
	a := NewID()
	time.Sleep(2 * time.Millisecond)
	b := NewID()

	if a == b {
		t.Error("ULID 不该重复")
	}
	if len(a) != 26 || len(b) != 26 {
		t.Errorf("ULID 长度应为 26, got %d / %d", len(a), len(b))
	}
	if a >= b {
		t.Errorf("先生成的 ULID 应字典序在前: %s >= %s", a, b)
	}
}
