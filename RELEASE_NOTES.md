# Release Notes

## v0.28.0 - 2026-06-30

### 新增

- 新增 simple run mode 资源默认值覆盖逻辑，未显式配置时自动使用更保守的数据库、Redis、调度和网关连接参数。
- 新增 simple run mode 下 usage record worker 与 auto scale 边界归一化，避免默认值组合产生无效区间。

### 改进

- simple run mode 默认关闭 Ops 实时监控，并将 Ops metrics 与渠道监控默认间隔调整为 300 秒，降低轻量部署后台压力。
- settings 初始化、公开配置读取和运行时解析统一按 run mode 使用同一组默认值，减少前后端展示与后端行为不一致。

### 修复

- 修正 simple run mode 下显式配置 worker 数时，隐式 auto scale 上限可能低于 worker 数的问题。
- 修正缺省 settings 解析始终按标准模式启用实时监控和 60 秒轮询的问题。

### 发布

- 更新版本号到 `0.28.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub/Railway 镜像渠道不随常规发布自动执行；如需要，单独走“镜像渠道发布”流程。
