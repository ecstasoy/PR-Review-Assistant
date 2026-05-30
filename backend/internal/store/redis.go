package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache 实现 Cache 接口，行为与 MemoryCache 等价但跨实例共享。
// 用途：v3 真部署多副本时 rate limit 计数 / SSE session 状态需要全局一致。
//
// 与 MemoryCache 的差异：
//   - Get 三返回值（value, ok, err）：redis.Nil 映射为 (nil, false, nil)，其他 err 透传
//   - Incr 用 Redis INCR + EXPIRE NX 复合保证首次设 TTL，已存在的 key 不重设
//   - 进程退出 / 重启数据保留（Redis 持久化）；MemoryCache 重启清零
//
// 默认连接池由 go-redis 管理（PoolSize=10*GOMAXPROCS），单机 Fly.io shared-cpu-1x 够用。
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 解析 URL（形如 redis://:password@host:6379/0）打开连接，
// ping 一次确保可达。失败时返 error，main 接线侧应降级到 MemoryCache。
func NewRedisCache(url string) (*RedisCache, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis parse url: %w", err)
	}
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisCache{client: client}, nil
}

// Close 关闭底层连接池；多次调用安全（go-redis 内部幂等）。
func (c *RedisCache) Close() error { return c.client.Close() }

// Get redis.Nil 视为 miss，其他 err 透传。
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}

// Set 写值并设 TTL；ttl=0 表示永久（Redis 命令省略 EX 参数）。
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Incr 原子递增 + 首次 SET 时设 TTL，已存在的不重设。
// 用 INCR + EXPIRE 两步实现；只有 INCR 返回 1 的创建者设置 TTL。
// ttl=0 时不设过期（等价 INCR 一句）。
func (c *RedisCache) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if ttl > 0 && n == 1 {
		// 仅首次（INCR 返回 1）设 TTL，避免每次请求都刷
		if _, err := c.client.Expire(ctx, key, ttl).Result(); err != nil {
			// EXPIRE 失败不致命（计数器仍工作）；记 warn 由调用方处理
			return n, fmt.Errorf("incr: set expire failed (count saved): %w", err)
		}
	}
	return n, nil
}

// Delete 立即移除；不存在的 key 不报错。
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Ping 健康检查；readiness handler 用。
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// 编译期断言 RedisCache 实现 Cache 接口
var _ Cache = (*RedisCache)(nil)
