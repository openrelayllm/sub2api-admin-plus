# Release Notes

## v0.17.0 - 2026-06-26

### 新增

- 新增供应商分组变更事件记录，支持追踪分组成员同步和供应商账号归属变化。
- 新增后台备份入口和相关路由，为运维侧备份管理提供独立页面。
- 扩展站点发现与注册流程的自动化处理能力，补充更多后台操作和任务状态信息。

### 改进

- Mihomo runtime 启动时显式使用每个运行槽位目录作为配置目录，并设置可写的 `HOME` / `XDG_CONFIG_HOME`。
- 代理运行槽位创建增加并发冲突重试，一键检测多个节点时会复用刚创建的空闲槽位。
- 调度、通知、动作推荐和供应商分组页面补充状态展示与接口字段。

### 修复

- 修复线上 systemd `ProtectSystem=strict` 下 Mihomo 默认写入 `/opt/sub2api/.config/mihomo` 失败，导致本地代理端口未监听的问题。
- 修复一键检测并发创建 runtime slot 时偶发 `idx_admin_plus_proxy_runtime_slots_key` 唯一键冲突的问题。
- 修复部分站点发现、扩展任务和通知流程的状态同步与测试覆盖缺口。

### 发布

- 更新版本号到 `0.17.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub/Railway 镜像渠道不随常规发布自动执行；如需要，单独走“镜像渠道发布”流程。
