package store

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// NewID 生成 26 字符 ULID（时间前缀 + 随机），按时间升序可排，且全局唯一。
// 用 crypto/rand 而非 math/rand 避免并发竞态以及可预测性。
func NewID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
