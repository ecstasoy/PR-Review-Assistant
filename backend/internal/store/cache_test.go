package store

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	c := NewMemoryCache(time.Minute)
	defer c.Close()
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

func TestMemoryCache_Miss(t *testing.T) {
	c := NewMemoryCache(time.Minute)
	defer c.Close()
	_, ok, err := c.Get(context.Background(), "missing")
	if err != nil || ok {
		t.Errorf("miss want ok=false err=nil, got ok=%v err=%v", ok, err)
	}
}

func TestMemoryCache_TTLExpires(t *testing.T) {
	c := NewMemoryCache(time.Minute)
	defer c.Close()
	ctx := context.Background()

	_ = c.Set(ctx, "ephemeral", []byte("x"), 10*time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	_, ok, _ := c.Get(ctx, "ephemeral")
	if ok {
		t.Errorf("ttl expired entry should be invisible")
	}
}

func TestMemoryCache_Incr(t *testing.T) {
	c := NewMemoryCache(time.Minute)
	defer c.Close()
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

func TestMemoryCache_Incr_TTL(t *testing.T) {
	c := NewMemoryCache(time.Minute)
	defer c.Close()
	ctx := context.Background()

	n, _ := c.Incr(ctx, "k", 10*time.Millisecond)
	if n != 1 {
		t.Fatalf("first incr should be 1, got %d", n)
	}
	time.Sleep(30 * time.Millisecond)
	// 过期后再 Incr 应该重新从 1 开始
	n, _ = c.Incr(ctx, "k", 10*time.Millisecond)
	if n != 1 {
		t.Errorf("post-expire incr should reset to 1, got %d", n)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	c := NewMemoryCache(time.Minute)
	defer c.Close()
	ctx := context.Background()

	_ = c.Set(ctx, "k", []byte("v"), 0)
	_ = c.Delete(ctx, "k")
	_, ok, _ := c.Get(ctx, "k")
	if ok {
		t.Errorf("delete failed: still ok")
	}
}

// 编译期断言 MemoryCache 实现 Cache 接口
var _ Cache = (*MemoryCache)(nil)
