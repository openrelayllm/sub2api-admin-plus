.PHONY: build build-backend build-frontend build-datamanagementd dev dev-backend dev-frontend e2e-local test test-backend test-frontend test-frontend-critical test-datamanagementd secret-scan

FRONTEND_CRITICAL_VITEST := \
	src/router/__tests__/admin-plus-routes.spec.ts \
	src/router/__tests__/title.spec.ts \
	src/views/admin/ops/components/__tests__/OpsOpenAITokenStatsCard.spec.ts \
	src/views/admin/ops/components/__tests__/OpsErrorLogTable.spec.ts \
	src/views/admin/ops/components/__tests__/OpsErrorScopeCharts.spec.ts \
	src/views/admin/ops/utils/__tests__/errorDetailResponse.spec.ts

# 一键编译前后端
build: build-backend build-frontend

# 编译后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build

# 编译前端（需要已安装依赖）
build-frontend:
	@pnpm --dir frontend run build

# 编译 datamanagementd（宿主机数据管理进程）
build-datamanagementd:
	@cd datamanagement && go build -o datamanagementd ./cmd/datamanagementd

# 运行测试（后端 + 前端）
test: test-backend test-frontend

dev:
	@bash scripts/start-dev.sh

dev-backend:
	@bash scripts/start-backend.sh

dev-frontend:
	@bash scripts/start-frontend.sh

e2e-local:
	@bash scripts/run-e2e-local.sh

test-backend:
	@$(MAKE) -C backend test

test-frontend:
	@pnpm --dir frontend run lint:check
	@pnpm --dir frontend run typecheck
	@$(MAKE) test-frontend-critical

test-frontend-critical:
	@pnpm --dir frontend exec vitest run $(FRONTEND_CRITICAL_VITEST)

test-datamanagementd:
	@cd datamanagement && go test ./...

secret-scan:
	@python3 tools/secret_scan.py
