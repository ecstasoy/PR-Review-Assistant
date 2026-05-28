.PHONY: help dev backend frontend install build test clean demo

# 默认 target：列出可用命令
help:
	@echo "可用 target："
	@echo "  install  装前后端依赖（go mod + pnpm install）"
	@echo "  dev      并发起前后端（Ctrl-C 同时杀掉两侧）"
	@echo "  backend  仅起 Go 后端 :8080"
	@echo "  frontend 仅起 Next.js 前端 :3000（需 Node ≥ 20）"
	@echo "  build    打产物（backend/bin/server + frontend/.next）"
	@echo "  test     跑后端 go test"
	@echo "  clean    清产物"
	@echo "  demo     打印用来本地试跑的公开 PR 列表"

install:
	cd backend && go mod tidy
	cd frontend && pnpm install

dev:
	@trap 'kill 0' EXIT INT TERM; \
	$(MAKE) backend & \
	$(MAKE) frontend & \
	wait

backend:
	cd backend && go run ./cmd/server

frontend:
	@node -v | grep -qE 'v(2[0-9]|[3-9][0-9])' || { echo "需要 Node ≥ 20（当前 $$(node -v)），请先 nvm use 20"; exit 1; }
	cd frontend && pnpm dev

build:
	cd backend && go build -o ./bin/server ./cmd/server
	cd frontend && pnpm build

test:
	cd backend && go test ./...

clean:
	rm -rf backend/bin frontend/.next

demo:
	@echo "本地试跑用的公开 PR："
	@echo "  https://github.com/golang/go/pull/1"
	@echo "  https://github.com/vercel/next.js/pull/100"
	@echo "  https://github.com/qodo-ai/pr-agent/pull/1000"
	@echo ""
	@echo "提示：LLM_PROVIDER=mock 可免 API key 跑全流程"
