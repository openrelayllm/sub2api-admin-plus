# Release Notes

## v0.41.0 - 2026-07-09

### 新增

- GitHub Release workflow 支持 `v*` tag 自动发布，统一生成 Linux 二进制资产、DockerHub 多架构镜像和 GHCR 多架构镜像。
- 新增独立的 Linux 二进制 GoReleaser 配置，CI Build Artifacts 只构建 `linux_amd64`、`linux_arm64` 和 `checksums.txt`，避免快照构建依赖镜像发布。
- Release workflow 支持可选 Railway 部署开关；默认不自动部署 Railway，只有启用 `RAILWAY_AUTO_DEPLOY=true` 或手动勾选 `deploy_railway` 时执行。
- 补充 P1/P2 发布验收 Runbook、上线后观察与回滚 Runbook、发布就绪快照和 A1-A12 验收记录模板。

### 改进

- Docker Compose 默认镜像切换为 `wutongci/sub2api-admin-plus:latest`，并支持通过 `ADMIN_PLUS_IMAGE=wutongci/sub2api-admin-plus:0.41.0` 固定版本。
- Docker 构建基础 Go 镜像升级到 `golang:1.26.5-alpine`，容器构建会从 `VERSION` 文件读取版本并写入二进制 ldflags。
- 后台版本信息将 `build_type` 收敛为 `source`、`release`、`container` 三种类型，容器部署继续在运行时识别。
- 管理端一键更新成功后，如果后端返回 `need_restart=true`，前端会自动调用重启流程，减少二进制替换后还需手动点击重启的步骤。
- 供应商架构文档补充接手者阅读顺序、P3 不实施边界、P1/P2 发布前人工验收门禁和上线后观察要求。

### 修复

- 修正后台更新成功响应的测试覆盖，确保成功下载并替换二进制时返回 `need_restart=true`。
- 修正文档中 P0/P1/P2/P3 阶段名混用的口径，明确账务、看板、历史重构专项阶段不等同于 supplier architecture 当前收口边界。

### 测试

- 增加系统更新成功后需要重启的 handler 回归测试。
- 补充 README/09/10/11/12/13 相对链接检查和 A1-A12 代码级证据文件存在性检查记录。

### 发布

- 更新版本号到 `0.41.0`。
- GitHub Release 保持 Linux-only 二进制资产：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- tag 发布同步发布 DockerHub 与 GHCR 多架构镜像：`0.41.0`、`latest`、`0.41` 和 `0`。
- Railway 默认不自动部署；如需部署，单独启用 Release workflow 的 `deploy_railway` 或仓库变量 `RAILWAY_AUTO_DEPLOY=true`。
- 裸机 systemd 部署继续使用 GitHub Release 二进制升级；容器部署通过拉取新镜像升级。
