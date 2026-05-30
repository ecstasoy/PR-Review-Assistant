package store

import (
	"context"
	"os"
	"testing"
	"time"
)

// redisTestCache 用 REDIS_TEST_URL；空时整 file 测试 skip
// CI 默认不跑（避免每个 PR 都启 Redis service）；本地：
//
//	docker run --rm -p 6379:6379 redis:7-alpine
//	export REDIS_TEST_URL=redis://localhost:6379/15
//	go test ./internal/store/ -run TestRedis -v
//
// 用 DB 15 避免污染本地用的 DB 0。
func redisTestCache(t *testing.T) *RedisCache {
	t.Helper()
	url := os.Getenv("REDIS_TEST_URL")
	if url == "" {
		t.Skip("REDIS_TEST_URL not set; skipping redis integration tests")
	}
	c, err := NewRedisCache(url)
	if err != nil {
		t.Fatalf("NewRedisCache: %v", err)
	}
	// 每次测试前清掉测试 DB，避免互相污染
	if err := c.client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestRedisCache_SetGet(t *testing.T) {
	c := redisTestCache(t)
	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, ok, err := c.Get(ctx, "k")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !ok || string(got) != "v" {
		t.Errorf("get k: ok=%v val=%s want ok=true val=v", ok, got)
	}
}

func TestRedisCache_Miss(t *testing.T) {
	c := redisTestCache(t)
	_, ok, err := c.Get(context.Background(), "missing")
	if err != nil || ok {
		t.Errorf("miss want ok=false err=nil, got ok=%v err=%v", ok, err)
	}
}

func TestRedisCache_TTLExpires(t *testing.T) {
	c := redisTestCache(t)
	ctx := context.Background()
	if err := c.Set(ctx, "ephemeral", []byte("x"), 50*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	_, ok, err := c.Get(ctx, "ephemeral")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if ok {
		t.Errorf("ttl expired entry should be invisible")
	}
}

func TestRedisCache_Incr(t *testing.T) {
	c := redisTestCache(t)
	ctx := context.Background()
	for i := 1; i <= 5; i++ {
		got, err := c.Incr(ctx, "counter", time.Minute)
		if err != nil {
			t.Fatalf("incr: %v", err)
		}
		if got != int64(i) {
			t.Errorf("incr round %d got=%d want=%d", i, got, i)
		}
	}
}

func TestRedisCache_Incr_TTL(t *testing.T) {
	c := redisTestCache(t)
	ctx := context.Background()
	n, _ := c.Incr(ctx, "k", 50*time.Millisecond)
	if n != 1 {
		t.Fatalf("first incr should be 1, got %d", n)
	}
	time.Sleep(100 * time.Millisecond)
	// 过期后再 Incr 应该重新从 1 开始
	n, _ = c.Incr(ctx, "k", 50*time.Millisecond)
	if n != 1 {
		t.Errorf("post-expire incr should reset to 1, got %d", n)
	}
}

func TestRedisCache_Delete(t *testing.T) {
	c := redisTestCache(t)
	ctx := context.Background()
	_ = c.Set(ctx, "k", []byte("v"), 0)
	_ = c.Delete(ctx, "k")
	_, ok, _ := c.Get(ctx, "k")
	if ok {
		t.Errorf("delete failed: still ok")
	}
}

func TestRedisCache_Ping(t *testing.T) {
	c := redisTestCache(t)
	if err := c.Ping(context.Background()); err != nil {
		t.Errorf("ping: %v", err)
	}
}

func TestRedisCache_InvalidURL(t *testing.T) {
	// 不依赖 Redis 实例；URL 解析就该 fail
	_, err := NewRedisCache("not-a-url")
	if err == nil {
		t.Errorf("expected parse err on invalid URL")
	}
}
