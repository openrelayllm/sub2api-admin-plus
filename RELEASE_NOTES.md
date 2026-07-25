# Release Notes

## v0.43.3 - 2026-07-25

### 修复

- 修复纯度检测 SSE 的 started/progress 中间报告把 `score_policy.dimensions` 序列化为 `null`，导致公开前端校验失败并错误显示 0 分的问题。
- 评分策略的 snake_case 与 camelCase 兼容字段现在都会稳定输出维度数组，最终渠道评分与 Detail 不再被中间事件截断。
- 公开前端同时兼容历史部署或代理缓存中的 `null` 数组值，并规范化为空数组后继续接收最终报告。

### 测试

- 新增后端进度事件 JSON 契约回归测试，覆盖 `score_policy` 和 `scorePolicy`。
- 新增前端 Zod 兼容回归与 progress -> report 流式页面回归，确认检测不会进入假失败或 0 分状态。
- Admin Plus 前端 34 个测试文件、196 项测试通过；公开页 6 个测试文件、31 项测试通过。

### 发布

- 更新版本号到 `0.43.3`。
- GitHub Release 只发布 Linux 资产：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- tag 发布同步发布 DockerHub 与 GHCR 多架构镜像：`0.43.3`、`latest`、`0.43` 和 `0`。
- 裸机 systemd 部署通过 `sub2apiplus upgrade -v v0.43.3` 升级。
