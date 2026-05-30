# 后端容器化：multi-stage，alpine builder + alpine runner
# go-sqlite3 是 CGO，distroless static 跑不动，所以走 alpine + musl 静态链接路线

# ============ builder ============
FROM golang:1.25-alpine AS builder

# CGO 工具链 + sqlite 头文件
RUN apk add --no-cache build-base sqlite-dev

WORKDIR /src

# 利用 docker cache：先复制 mod 文件单独下载依赖
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 静态链接 musl，binary 自带依赖，runner 不需要 libc
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags '-s -w -linkmode external -extldflags "-static"' \
    -o /out/server ./cmd/server

# ============ runner ============
FROM alpine:3.20

# CA certs 给 https 用；tzdata 给 logging 用本地时区
RUN apk add --no-cache ca-certificates tzdata

# 非 root 用户
RUN addgroup -g 1000 app && adduser -u 1000 -G app -D -h /app app

# 数据目录在切换用户前创建并赋权，确保挂卷后 app 进程可写
RUN mkdir -p /app/data && chown app:app /app/data

USER app
WORKDIR /app

COPY --from=builder /out/server /app/server

# SQLite 文件挂卷
VOLUME ["/app/data"]
ENV SQLITE_PATH=/app/data/reviews.db
ENV PORT=8080

EXPOSE 8080

# 健康检查走 /api/health；30s 一次足够
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/server"]
