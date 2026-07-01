# Release Notes

## v0.33.0 - 2026-07-02

### 改进

- NewAPI 成本、充值和兑换记录改用管理员接口分页读取，支持读取完整历史数据。
- Sub2API 用量成本优先读取管理员用量接口，并按页拉取、按时间窗口过滤；普通用户会话自动回退用户用量接口。
- NewAPI 直接登录和注册会话写入角色/状态信息，成本历史读取会在普通用户会话下给出明确的管理员会话要求。
- 成本回补运行列表和 Step 明细支持分页加载，避免大批量历史任务挤爆单页响应。
- Redis 中的调度账号缓存和代理延迟缓存增加 TTL，减少长期 stale cache。

### 新增

- 成本回补历史支持物理删除单个 Run，自动清理对应 step 和 attempt。
- 成本回补历史支持按任务类型批量清空已结束 Run，queued/running 任务会被保留。
- 前端成本对账页面新增取消 Run、删除当前 Run、删除列表 Run、清空已结束历史和 Run/Step 翻页操作。
- NewAPI 支持兑换码/权益记录读取，并对兑换码做脱敏指纹和尾号展示。

### 修复

- 取消或删除调度 Run 后，worker 不再领取该 Run 的后续 step，运行中的 step 会感知 Run 状态并停止继续处理。
- 已非 running 状态的 step 不再被完成写回，避免取消后的旧 worker 覆盖最终状态。
- NewAPI 管理员会话不足时统一进入 manual required，并给出重新一键登录管理员/root 账号的处理建议。

### 测试

- 增加 NewAPI 管理员历史读取、兑换记录读取、Sub2API 管理员/用户用量分页、调度 Run 删除和取消后不再领取 step 的回归测试。

### 发布

- 更新版本号到 `0.33.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub/Railway 镜像渠道不随常规发布自动执行；如需要，单独走镜像渠道发布流程。
