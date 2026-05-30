// Package store 持久化评审：按 head SHA 缓存避免重复花钱，同时撑 /history 页。
package store

import (
	"context"
	"encoding/json"
	"time"
)

// Record 一条缓存评审
type Record struct {
	ID        string          // ulid
	UserID    *string         // v1 永远 nil；v2 OAuth 后填
	Owner     string
	Repo      string
	PRNumber  int
	HeadSHA   string
	Payload   json.RawMessage // 序列化的 review.Result(JSON)
	CreatedAt time.Time
}

// Store 缓存 + 历史，统一在一个接口里
type Store interface {
	Get(ctx context.Context, owner, repo string, pr int, headSHA string) (*Record, error)
	Put(ctx context.Context, r *Record) error
	List(ctx context.Context, userID *string, limit int) ([]*Record, error)
	GetByID(ctx context.Context, id string) (*Record, error)
	// Ping 健康检查；ctx 带 timeout。SQLite 走 db.PingContext，Postgres 同理。
	Ping(ctx context.Context) error
}
