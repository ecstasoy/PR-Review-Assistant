#!/bin/sh
# 容器入口：后台跑全仓 RAG 预索引 + 前台启 server
# 设计要点：
#   - 索引在后台，server 立即可用（避免阻塞 readiness check）
#   - RAG_SCOPE 未设 / 源码缺失 / 索引失败 → 仅 log，server 照常启动
#   - ON CONFLICT 让索引幂等：每次部署都 re-run 也只覆盖不重复 embed 同内容

set -e

if [ -n "$RAG_SCOPE" ] && [ -d /app/src ]; then
    (
        echo "[entrypoint] background indexrepo: scope=$RAG_SCOPE dir=/app/src db=${RAG_DB_PATH:-/data/rag.db}"
        if /app/indexrepo \
            --scope "$RAG_SCOPE" \
            --dir /app/src \
            --db "${RAG_DB_PATH:-/data/rag.db}"; then
            echo "[entrypoint] indexrepo finished"
        else
            echo "[entrypoint] indexrepo failed (exit $?); RAG falls back to PR-indexed chunks only"
        fi
    ) &
else
    echo "[entrypoint] RAG_SCOPE not set or /app/src missing; skipping full-repo pre-index"
fi

echo "[entrypoint] exec server"
exec /app/server
