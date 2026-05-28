package store

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore 单文件 SQLite 实现 Store。
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore 打开或创建数据库。PR #13 加 schema 自动迁移 + WAL
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

// Close 关闭底层 db handle
func (s *SQLiteStore) Close() error { return s.db.Close() }

// Get 实现 Store
func (s *SQLiteStore) Get(ctx context.Context, owner, repo string, pr int, headSHA string) (*Record, error) {
	return nil, errors.New("SQLiteStore.Get: not implemented yet")
}

// Put 实现 Store
func (s *SQLiteStore) Put(ctx context.Context, r *Record) error {
	return errors.New("SQLiteStore.Put: not implemented yet")
}

// List 实现 Store
func (s *SQLiteStore) List(ctx context.Context, userID *string, limit int) ([]*Record, error) {
	return nil, errors.New("SQLiteStore.List: not implemented yet")
}

// GetByID 实现 Store
func (s *SQLiteStore) GetByID(ctx context.Context, id string) (*Record, error) {
	return nil, errors.New("SQLiteStore.GetByID: not implemented yet")
}
