.PHONY: build build-backend build-noembed build-frontend build-docs test test-backend test-frontend test-frontend-critical test-e2e

FRONTEND_CRITICAL_VITEST := \
	src/views/auth/__tests__/LinuxDoCallbackView.spec.ts \
	src/views/auth/__tests__/WechatCallbackView.spec.ts \
	src/views/user/__tests__/PaymentView.spec.ts \
	src/views/user/__tests__/PaymentResultView.spec.ts \
	src/components/user/profile/__tests__/ProfileInfoCard.spec.ts \
	src/views/admin/__tests__/SettingsView.spec.ts \
	src/utils/__tests__/documentationUrl.spec.ts \
	src/utils/__tests__/favicon.spec.ts

# 一键编译嵌入式发布产物。使用递归 make 保证即使传入 -j，顺序仍然固定。
build:
	@$(MAKE) build-frontend
	@$(MAKE) build-docs
	@$(MAKE) build-backend

# 编译带嵌入资源的后端（复用 backend/Makefile）
build-backend:
	@$(MAKE) -C backend build-embed

# 显式的无嵌入开发构建，不可用于发布。
build-noembed:
	@$(MAKE) -C backend build-noembed

# 编译前端（需要已安装依赖）
build-frontend:
	@cd frontend && corepack pnpm run build

build-docs:
	@npm --prefix docs-site run build
	@npm --prefix docs-site run verify:build
	@rm -rf backend/internal/web/dist/tutorial-docs
	@mkdir -p backend/internal/web/dist/tutorial-docs
	@cp -a docs-site/dist/. backend/internal/web/dist/tutorial-docs/
	@test -f backend/internal/web/dist/tutorial-docs/index.html
	@grep -q '/tutorial-docs/assets/' backend/internal/web/dist/tutorial-docs/index.html

# 运行测试（后端 + 前端）
test: test-backend test-frontend

test-backend:
	@$(MAKE) -C backend test

test-e2e:
	@$(MAKE) -C backend test-e2e

test-frontend:
	@cd frontend && corepack pnpm run lint:check
	@cd frontend && corepack pnpm run typecheck
	@cd frontend && corepack pnpm run typecheck:test
	@cd frontend && corepack pnpm run test:run

test-frontend-critical:
	@cd frontend && corepack pnpm exec vitest run $(FRONTEND_CRITICAL_VITEST)
