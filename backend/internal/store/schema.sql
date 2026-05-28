-- v1 schema；v2 加 users / comments 表不破坏现有。

CREATE TABLE IF NOT EXISTS reviews (
    id          TEXT PRIMARY KEY,
    user_id     TEXT,                       -- v1 可空；v2 OAuth 后填
    owner       TEXT NOT NULL,
    repo        TEXT NOT NULL,
    pr_number   INTEGER NOT NULL,
    head_sha    TEXT NOT NULL,
    payload     TEXT NOT NULL,              -- 序列化的 review.Result
    created_at  INTEGER NOT NULL,
    UNIQUE(owner, repo, pr_number, head_sha)
);

CREATE INDEX IF NOT EXISTS idx_reviews_user
    ON reviews(user_id, created_at DESC);
