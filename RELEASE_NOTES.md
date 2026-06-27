# Release Notes

## v0.22.0 - 2026-06-27

### 新增

- 新增 ProxyAI Developer API 纯度检测入口，支持 API Key 鉴权的普通检测与流式检测。
- 新增纯度检测报告访问模式与计费模式字段，区分 Web、Developer API 和后台账号来源。

### 改进

- Web 公开纯度检测继续使用 Turnstile 与限流保护，Developer API 入口改为 API Key 计量鉴权。
- 公开 ProxyAI CORS 预检允许 `Authorization`、`X-API-Key` 和 `X-ProxyAI-Key`，便于独立页面调用 Developer API。
- 旧版 `/api/v1/public/proxyai/purity/checks*` 路径收敛到 API Key 鉴权入口，避免绕过 Developer API 权限模型。

### 修复

- 补充 ProxyAI 公开路由、CORS、fail-closed 鉴权和纯度报告模式字段测试，降低接口回归风险。

### 发布

- 更新版本号到 `0.22.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub/Railway 镜像渠道不随常规发布自动执行；如需要，单独走“镜像渠道发布”流程。
