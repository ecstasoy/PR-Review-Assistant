package store

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// postgresTestStore 取 PG_TEST_URL；空时整个 file 的测试 skip
// CI 默认不跑（避免要求每个 PR 都启 PG service）；本地 docker compose 起 PG 后 export 即可：
//
//	export PG_TEST_URL=postgres://lgtm:lgtm@localhost:5432/lgtm?sslmode=disable
//	go test ./internal/store/ -run TestPostgres -v
func postgresTestStore(t *testing.T) *PostgresStore {
	t.Helper()
	dsn := os.Getenv("PG_TEST_URL")
	if dsn == "" {
		t.Skip("PG_TEST_URL not set; skipping postgres integration tests")
	}
	s, err := NewPostgresStore(dsn)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}
	// 每次测试前清表，避免互相污染
	if _, err := s.db.ExecContext(context.Background(), "TRUNCATE TABLE reviews"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func pgSampleRecord(headSHA string) *Record {
	return &Record{
		ID:        NewID(),
		Owner:     "golang",
		Repo:      "go",
		PRNumber:  42,
		HeadSHA:   headSHA,
		Payload:   json.RawMessage(`{"summary":"hello"}`),
		CreatedAt: time.Unix(1700_000_000, 0).UTC(),
	}
}

func TestPostgresStore_PutGet_RoundTrip(t *testing.T) {
	s := postgresTestStore(t)
	ctx := context.Background()
	rec := pgSampleRecord("sha-roundtrip")
	if err := s.Put(ctx, rec); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := s.Get(ctx, rec.Owner, rec.Repo, rec.PRNumber, rec.HeadSHA)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatalf("get miss; want record")
	}
	if got.ID != rec.ID || string(got.Payload) != string(rec.Payload) {
		t.Errorf("roundtrip mismatch: got=%+v want=%+v", got, rec)
	}
}

func TestPostgresStore_Get_MissReturnsNilNilNotError(t *testing.T) {
	s := postgresTestStore(t)
	got, err := s.Get(context.Background(), "no", "such", 1, "sha")
	if got != nil || err != nil {
		t.Errorf("want (nil, nil), got (%v, %v)", got, err)
	}
}

func TestPostgresStore_Ping(t *testing.T) {
	s := postgresTestStore(t)
	if err := s.Ping(context.Background()); err != nil {
		t.Errorf("ping: %v", err)
	}
}

func TestPostgresStore_Put_SameSHAPreservesID(t *testing.T) {
	s := postgresTestStore(t)
	ctx := context.Background()
	rec1 := pgSampleRecord("sha-same")
	if err := s.Put(ctx, rec1); err != nil {
		t.Fatalf("put1: %v", err)
	}
	rec2 := pgSampleRecord("sha-same")
	rec2.Payload = json.RawMessage(`{"summary":"updated"}`)
	if err := s.Put(ctx, rec2); err != nil {
		t.Fatalf("put2: %v", err)
	}
	got, err := s.Get(ctx, rec1.Owner, rec1.Repo, rec1.PRNumber, rec1.HeadSHA)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatalf("get miss; want record")
	}
	if got.ID != rec1.ID {
		t.Errorf("ON CONFLICT 应保留原 ID, got=%s want=%s", got.ID, rec1.ID)
	}
	if string(got.Payload) != `{"summary":"updated"}` {
		t.Errorf("payload 未刷新: %s", got.Payload)
	}
}

func TestPostgresStore_List_OrdersByCreatedAtDesc(t *testing.T) {
	s := postgresTestStore(t)
	ctx := context.Background()
	r1 := pgSampleRecord("sha-old")
	r1.CreatedAt = time.Unix(1000, 0).UTC()
	r1.HeadSHA = "sha-old"
	r2 := pgSampleRecord("sha-new")
	r2.CreatedAt = time.Unix(2000, 0).UTC()
	r2.HeadSHA = "sha-new"
	if err := s.Put(ctx, r1); err != nil {
		t.Fatalf("put r1: %v", err)
	}
	if err := s.Put(ctx, r2); err != nil {
		t.Fatalf("put r2: %v", err)
	}
	list, err := s.List(ctx, nil, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2, got %d", len(list))
	}
	if list[0].HeadSHA != "sha-new" {
		t.Errorf("最新一条应排首位; got %s", list[0].HeadSHA)
	}
}

func TestPostgresStore_Put_EmptyIDRejected(t *testing.T) {
	s := postgresTestStore(t)
	r := pgSampleRecord("sha")
	r.ID = ""
	if err := s.Put(context.Background(), r); err == nil {
		t.Errorf("空 ID 应被拒绝")
	}
}

// 编译期断言 PostgresStore 实现 Store 接口
var _ Store = (*PostgresStore)(nil)
