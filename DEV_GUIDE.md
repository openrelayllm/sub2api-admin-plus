# Sub2API Admin Plus 开发指南

本文档记录 `sub2api-admin-plus` MVP 0 基线的开发约束和常用命令。

## 项目定位

`sub2api-admin-plus` 是基于 Sub2API 代码库复制出来的自动化运营扩展系统。

MVP 0 目标不是立即重构业务，而是先得到一个完整、可运行、风格一致的 Sub2API 克隆基线，之后再逐步清理和开发运营业务。

## 基线来源

| 项目 | 值 |
|------|----|
| 上游本地路径 | `/Users/coso/Documents/dev/go/sub2api` |
| 复制来源 commit | `4a5665da5b2c6b83c4597844ea6e573746c821b1` |
| 当前项目路径 | `/Users/coso/Documents/dev/ai/openrelayllm/sub2api-admin-plus` |

## 技术栈

| 模块 | 技术 |
|------|------|
| 后端 | Go, Gin, Ent |
| 前端 | Vue 3, Vite, TailwindCSS, pnpm |
| 数据库 | PostgreSQL |
| 缓存 | Redis |

## MVP 0 约束

- 不修改 `/Users/coso/Documents/dev/go/sub2api`。
- 保留 Sub2API 前后端架构和 UI 设计。
- 保留 Sub2API 后端 module 路径，暂不做全仓 import 迁移。
- 先保证克隆基线可构建，再逐步做业务清理。
- Admin Plus 私有业务进入独立模块，不直接写入上游复制区的核心逻辑。
- 文档优先维护在 `docs/`。

## 常用命令

后端：

```bash
cd backend
go test ./...
go build -o bin/server ./cmd/server
```

前端：

```bash
cd frontend
pnpm install
pnpm run typecheck
pnpm run build
```

整体：

```bash
make build
make test-backend
make test-frontend
```

## 当前注意事项

- `backend/go.mod` 要求 Go `1.26.4`。本机如果是 Go `1.24.3`，默认 Go toolchain 可自动下载并使用 `1.26.4`；如果环境禁用了 toolchain，则需要手动安装或切换 Go 版本。
- 前端依赖使用 pnpm，不要用 npm 生成 lockfile。
- `docs/sub2api-admin-plus-prd.md` 是产品边界来源。
- `docs/code-structure.md` 是后续代码拆分和模块落地来源。
