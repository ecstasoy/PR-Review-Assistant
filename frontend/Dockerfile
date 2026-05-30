# 前端容器化：multi-stage，Next.js standalone output 让 runner 体积最小
# next.config.ts 需要 output: 'standalone'，下文 docker-compose 会通过 env 切

# ============ deps ============
FROM node:20-alpine AS deps
RUN corepack enable && corepack prepare pnpm@10.33.4 --activate
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# ============ builder ============
FROM node:20-alpine AS builder
RUN corepack enable && corepack prepare pnpm@10.33.4 --activate
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Next 15 standalone output；运行 next build
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm build

# ============ runner ============
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

# 非 root
RUN addgroup -g 1000 -S app && adduser -u 1000 -S app -G app
USER app

# Next standalone 输出包：.next/standalone（含 minimal server.js + deps）+ .next/static（静态资源）+ public
COPY --from=builder --chown=app:app /app/.next/standalone ./
COPY --from=builder --chown=app:app /app/.next/static ./.next/static
COPY --from=builder --chown=app:app /app/public ./public

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:3000 >/dev/null || exit 1

CMD ["node", "server.js"]
