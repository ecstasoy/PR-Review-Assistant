-- Postgres schema —— 与 SQLite 同结构，差异：BYTEA / BIGINT / 部分索引语法略不同
-- v1 reviews 表；v2 加 users / comments / vector chunks 不破坏现有

CREATE TABLE IF NOT EXISTS reviews (
    id          TEXT PRIMARY KEY,
    user_id     TEXT,                       -- v1 可空；v2 OAuth 后填
    owner       TEXT NOT NULL,
    repo        TEXT NOT NULL,
    pr_number   INTEGER NOT NULL,
    head_sha    TEXT NOT NULL,
    payload     BYTEA NOT NULL,             -- 序列化后的 review.Result 字节数据
    created_at  BIGINT NOT NULL             -- Unix 时间戳（纳秒）
);

-- 公开评审（无 user_id）的唯一约束：同 (owner,repo,pr,head_sha) 只一条
CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_public_unique
    ON reviews(owner, repo, pr_number, head_sha)
    WHERE user_id IS NULL;

-- 用户绑定评审的唯一约束（v2 用）
CREATE UNIQUE INDEX IF NOT EXISTS idx_reviews_user_unique
    ON reviews(user_id, owner, repo, pr_number, head_sha)
    WHERE user_id IS NOT NULL;

-- /history 列表的索引：按 (user_id, created_at DESC) 排
CREATE INDEX IF NOT EXISTS idx_reviews_user
    ON reviews(user_id, created_at DESC);
