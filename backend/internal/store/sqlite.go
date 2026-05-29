package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// SQLiteStore 单文件 SQLite 实现 Store。
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore 打开或创建数据库，应用 schema（CREATE IF NOT EXISTS 幂等），再返回。
// path 传 ":memory:" 即可获得纯内存库（测试用）；该 DSN 需固定单连接避免 schema 丢失。
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(context.Background(), schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

// Close 关闭底层 db handle。
func (s *SQLiteStore) Close() error { return s.db.Close() }

// Get 按 (owner, repo, pr_number, head_sha) 查缓存；未命中返 (nil, nil) 而非 error。
func (s *SQLiteStore) Get(ctx context.Context, owner, repo string, pr int, headSHA string) (*Record, error) {
	const q = `SELECT id, user_id, owner, repo, pr_number, head_sha, payload, created_at
	           FROM reviews
	           WHERE owner = ? AND repo = ? AND pr_number = ? AND head_sha = ? AND user_id IS NULL
	           LIMIT 1`
	row := s.db.QueryRowContext(ctx, q, owner, repo, pr, headSHA)
	return scanRecord(row)
}

// GetByID 按主键查；未命中返 (nil, nil)。
func (s *SQLiteStore) GetByID(ctx context.Context, id string) (*Record, error) {
	const q = `SELECT id, user_id, owner, repo, pr_number, head_sha, payload, created_at
	           FROM reviews WHERE id = ? LIMIT 1`
	row := s.db.QueryRowContext(ctx, q, id)
	return scanRecord(row)
}

// Put 写入或更新；冲突时按 (owner, repo, pr_number, head_sha) 复用既有 id 仅刷新 payload + 时间，
// 让前端的深链 id 在同 PR 重评后保持稳定。
// r.ID 为空时调用方应预先 store.NewID() 填好（caller 持有 ID 才能立即写回 SSE）。
func (s *SQLiteStore) Put(ctx context.Context, r *Record) error {
	if r.ID == "" {
		return errors.New("Record.ID is empty; generate with store.NewID() first")
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	} else {
		r.CreatedAt = r.CreatedAt.UTC()
	}
	const (
		qPublic = `INSERT INTO reviews (id, user_id, owner, repo, pr_number, head_sha, payload, created_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	           ON CONFLICT(owner, repo, pr_number, head_sha) WHERE user_id IS NULL
	           DO UPDATE SET payload = excluded.payload, created_at = excluded.created_at`
		qUser = `INSERT INTO reviews (id, user_id, owner, repo, pr_number, head_sha, payload, created_at)
	           VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	           ON CONFLICT(user_id, owner, repo, pr_number, head_sha) WHERE user_id IS NOT NULL
	           DO UPDATE SET payload = excluded.payload, created_at = excluded.created_at`
	)
	q := qPublic
	if r.UserID != nil {
		q = qUser
	}
	_, err := s.db.ExecContext(ctx, q,
		r.ID, r.UserID, r.Owner, r.Repo, r.PRNumber, r.HeadSHA, []byte(r.Payload), r.CreatedAt.UnixNano(),
	)
	return err
}

// List 按 user_id 过滤（nil → user_id IS NULL），按 created_at DESC 排序，最多 limit 条。
// limit ≤ 0 时强制 50（避免一次拉爆）。
func (s *SQLiteStore) List(ctx context.Context, userID *string, limit int) ([]*Record, error) {
	if limit <= 0 {
		limit = 50
	}
	const (
		qAll = `SELECT id, user_id, owner, repo, pr_number, head_sha, payload, created_at
		         FROM reviews WHERE user_id IS NULL ORDER BY created_at DESC, rowid DESC LIMIT ?`
		qUser = `SELECT id, user_id, owner, repo, pr_number, head_sha, payload, created_at
		         FROM reviews WHERE user_id = ? ORDER BY created_at DESC, rowid DESC LIMIT ?`
	)
	var (
		rows *sql.Rows
		err  error
	)
	if userID == nil {
		rows, err = s.db.QueryContext(ctx, qAll, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, qUser, *userID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Record
	for rows.Next() {
		r, err := scanRecordRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// scanRecord 单行 QueryRow 扫描；未命中转 (nil, nil)。
func scanRecord(row *sql.Row) (*Record, error) {
	var (
		r       Record
		payload []byte
		ts      int64
		userID  sql.NullString
	)
	err := row.Scan(&r.ID, &userID, &r.Owner, &r.Repo, &r.PRNumber, &r.HeadSHA, &payload, &ts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		s := userID.String
		r.UserID = &s
	}
	r.Payload = payload
	r.CreatedAt = time.Unix(0, ts).UTC()
	return &r, nil
}

// scanRecordRows 多行 Rows 扫描；调用方在循环里用。
func scanRecordRows(rows *sql.Rows) (*Record, error) {
	var (
		r       Record
		payload []byte
		ts      int64
		userID  sql.NullString
	)
	if err := rows.Scan(&r.ID, &userID, &r.Owner, &r.Repo, &r.PRNumber, &r.HeadSHA, &payload, &ts); err != nil {
		return nil, err
	}
	if userID.Valid {
		s := userID.String
		r.UserID = &s
	}
	r.Payload = payload
	r.CreatedAt = time.Unix(0, ts).UTC()
	return &r, nil
}
