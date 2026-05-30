package index

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteRetriever 用单 SQLite 文件存 chunk + embedding；
// 检索用 brute-force cosine（SELECT 所有 → 内存计算）。
//
// 为啥不用 sqlite-vss 扩展：
//   - 装载需要 .so / .dll；Dockerfile + macOS / 兼容性额外 30 min 调试预算
//   - demo 量级（单仓 < 10K chunks）brute-force 已经 sub-second
//   - v3 万级 chunk 时再上 sqlite-vss / pgvector / Qdrant
//
// 表结构 chunks(scope, path, idx, content, embedding BLOB)；按 (scope, path, idx) 唯一。
type SQLiteRetriever struct {
	db       *sql.DB
	embedder Embedder
}

// NewSQLiteRetriever 打开（或创建）chunks 表；embedder 用来给 Upsert / Retrieve 转向量。
// path = ":memory:" 测试用；生产传文件路径如 ./data/rag.db
func NewSQLiteRetriever(path string, embedder Embedder) (*SQLiteRetriever, error) {
	if embedder == nil {
		return nil, fmt.Errorf("retriever: embedder is nil")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("retriever open: %w", err)
	}
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("retriever ping: %w", err)
	}
	if _, err := db.ExecContext(context.Background(), retrieverSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("retriever apply schema: %w", err)
	}
	return &SQLiteRetriever{db: db, embedder: embedder}, nil
}

const retrieverSchema = `
CREATE TABLE IF NOT EXISTS chunks (
    scope     TEXT NOT NULL,
    path      TEXT NOT NULL,
    idx       INTEGER NOT NULL,
    content   TEXT NOT NULL,
    embedding BLOB NOT NULL,
    PRIMARY KEY (scope, path, idx)
);

CREATE INDEX IF NOT EXISTS idx_chunks_scope ON chunks(scope);
`

// Close 关闭底层 db。
func (r *SQLiteRetriever) Close() error { return r.db.Close() }

// Chunk 单个待索引文本片段
type Chunk struct {
	Path    string
	Idx     int    // 同 path 下的序号；切大文件用
	Content string // 实际文本内容
}

// UpsertMany 批量编码并写入；同 (scope, path, idx) 会覆盖。
// 失败整批返 err（半部分写入也算成功，下次重跑用 ON CONFLICT 自动去重）。
func (r *SQLiteRetriever) UpsertMany(ctx context.Context, scope string, chunks []Chunk) error {
	if len(chunks) == 0 {
		return nil
	}
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Content
	}
	vecs, err := r.embedder.Embed(ctx, texts)
	if err != nil {
		return fmt.Errorf("retriever upsert embed: %w", err)
	}
	if len(vecs) != len(chunks) {
		return fmt.Errorf("retriever upsert: embed returned %d vectors for %d chunks", len(vecs), len(chunks))
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	const q = `INSERT INTO chunks (scope, path, idx, content, embedding)
	           VALUES (?, ?, ?, ?, ?)
	           ON CONFLICT(scope, path, idx)
	           DO UPDATE SET content = excluded.content, embedding = excluded.embedding`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, c := range chunks {
		blob := encodeVec(vecs[i])
		if _, err := stmt.ExecContext(ctx, scope, c.Path, c.Idx, c.Content, blob); err != nil {
			return fmt.Errorf("retriever upsert exec [%s:%d]: %w", c.Path, c.Idx, err)
		}
	}
	return tx.Commit()
}

// Retrieve embed query → SELECT scope 内所有 chunks → 内存 cosine 排序 → top-K
func (r *SQLiteRetriever) Retrieve(ctx context.Context, scope, query string, k int) ([]Reference, error) {
	if k <= 0 {
		k = 4
	}
	if query == "" {
		return nil, nil
	}
	qv, err := r.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("retriever query embed: %w", err)
	}
	if len(qv) != 1 {
		return nil, fmt.Errorf("retriever: expected 1 query vec, got %d", len(qv))
	}
	queryVec := qv[0]

	// 全表 select 同 scope；万级以下 brute-force OK
	const q = `SELECT path, idx, content, embedding FROM chunks WHERE scope = ?`
	rows, err := r.db.QueryContext(ctx, q, scope)
	if err != nil {
		return nil, fmt.Errorf("retriever query: %w", err)
	}
	defer rows.Close()

	type scored struct {
		path    string
		idx     int
		content string
		score   float32
	}
	var hits []scored
	for rows.Next() {
		var (
			path, content string
			idx           int
			blob          []byte
		)
		if err := rows.Scan(&path, &idx, &content, &blob); err != nil {
			return nil, err
		}
		vec, err := decodeVec(blob)
		if err != nil {
			continue // 跳过损坏 row
		}
		s := cosineSim(queryVec, vec)
		hits = append(hits, scored{path, idx, content, s})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if len(hits) > k {
		hits = hits[:k]
	}
	out := make([]Reference, 0, len(hits))
	for _, h := range hits {
		out = append(out, Reference{
			File:    h.path,
			Snippet: h.content,
			Reason:  fmt.Sprintf("cosine=%.3f", h.score),
		})
	}
	return out, nil
}

// encodeVec [N]float32 → BLOB（小端 32-bit float 平铺）
func encodeVec(v []float32) []byte {
	buf := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

// decodeVec BLOB → [N]float32
func decodeVec(b []byte) ([]float32, error) {
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("decodeVec: length %d not multiple of 4", len(b))
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out, nil
}

// cosineSim 余弦相似度；假设两向量同维 + 已 L2-归一化（Embedder 约定输出），
// 直接 dot product 即可；未归一化时退化为 dot 不准确但仍可比较。
func cosineSim(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot float32
	for i := range a {
		dot += a[i] * b[i]
	}
	return dot
}

// 编译期断言
var _ Retriever = (*SQLiteRetriever)(nil)
