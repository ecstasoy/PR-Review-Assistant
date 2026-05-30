package store

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Cache 临时 KV，不持久。
// 用途：rate limit 计数 / SSE session 共享 / cross-instance state（v3 切 Redis 后真跨实例）。
// 与 Store 互补：Store 持久化业务记录；Cache 装易丢的 / 带 TTL 的状态。
type Cache interface {
	// Get 返 (value, ok, err)。ok=false 表示不存在或已过期，err 仅在底层故障时非 nil。
	Get(ctx context.Context, key string) (value []byte, ok bool, err error)
	// Set 写值并设 TTL；ttl=0 表示永久（直到进程退出 / 显式 Delete）。
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Incr 原子递增（不存在时从 0 起）；返回递增后的值。
	// 首次 Set 时若 ttl > 0 则同步设过期；已存在的 key 不重设 TTL。
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
	// Delete 立即删除。
	Delete(ctx context.Context, key string) error
}

// ErrCacheClosed 用 ctx.Done 通常足够；这个 error 是给未来 Redis 连接断开 / 重连失败用的占位。
var ErrCacheClosed = errors.New("cache: closed")

// ErrIncrOnNonCounter is returned when Incr is called on a key that was written via Set.
var ErrIncrOnNonCounter = errors.New("cache: Incr called on non-counter key")

// MemoryCache v1 内存实现。
//
// 特性：
//   - sync.Map：粗粒度并发安全；写多读多都还行；千 QPS 没压力
//   - 后台 goroutine 周期扫过期项；扫描间隔 = sweepInterval
//   - 进程重启即清空（语义上跟 ephemeral 一致）
//
// v3 Redis 实现来后业务代码改 NewRedisCache(...) 即可，无需改 handler。
type MemoryCache struct {
	items          sync.Map // map[string]*cacheItem
	sweepInterval  time.Duration
	stopOnce       sync.Once
	stop           chan struct{}
	incrMu         sync.Mutex // Incr 路径需要 read-modify-write 原子，比 sync.Map 自身更紧
}

type cacheItem struct {
	value     []byte
	intValue  int64
	isCounter bool
	expiresAt time.Time // 零值 = 永久
}

// NewMemoryCache 起一个内存缓存；sweepInterval 0 用默认 1 分钟。
func NewMemoryCache(sweepInterval time.Duration) *MemoryCache {
	if sweepInterval <= 0 {
		sweepInterval = time.Minute
	}
	c := &MemoryCache{
		sweepInterval: sweepInterval,
		stop:          make(chan struct{}),
	}
	go c.sweepLoop()
	return c
}

// Close 停止后台扫描；多次调用安全。
func (c *MemoryCache) Close() error {
	c.stopOnce.Do(func() { close(c.stop) })
	return nil
}

func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := c.items.Load(key)
	if !ok {
		return nil, false, nil
	}
	it := v.(*cacheItem)
	if !it.expiresAt.IsZero() && time.Now().After(it.expiresAt) {
		c.items.Delete(key)
		return nil, false, nil
	}
	cp := make([]byte, len(it.value))
	copy(cp, it.value)
	return cp, true, nil
}

func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	cp := make([]byte, len(value))
	copy(cp, value)
	it := &cacheItem{value: cp}
	if ttl > 0 {
		it.expiresAt = time.Now().Add(ttl)
	}
	c.items.Store(key, it)
	return nil
}

func (c *MemoryCache) Incr(_ context.Context, key string, ttl time.Duration) (int64, error) {
	c.incrMu.Lock()
	defer c.incrMu.Unlock()
	now := time.Now()
	if v, ok := c.items.Load(key); ok {
		it := v.(*cacheItem)
		if it.expiresAt.IsZero() || now.Before(it.expiresAt) {
			if !it.isCounter {
				return 0, ErrIncrOnNonCounter
			}
			it.intValue++
			it.isCounter = true
			c.items.Store(key, it)
			return it.intValue, nil
		}
		// 过期：当作不存在
	}
	it := &cacheItem{intValue: 1, isCounter: true}
	if ttl > 0 {
		it.expiresAt = now.Add(ttl)
	}
	c.items.Store(key, it)
	return 1, nil
}

func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.items.Delete(key)
	return nil
}

// sweepLoop 周期清过期项；进程退出时停。
// 简单 O(n) 扫；千级 key 仍秒级完成。v3 切 Redis 后此循环可去掉。
func (c *MemoryCache) sweepLoop() {
	ticker := time.NewTicker(c.sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			now := time.Now()
			c.items.Range(func(k, v any) bool {
				it := v.(*cacheItem)
				if !it.expiresAt.IsZero() && now.After(it.expiresAt) {
					c.items.Delete(k)
				}
				return true
			})
		}
	}
}
