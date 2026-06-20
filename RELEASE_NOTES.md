# Release Notes

## v0.6.0 - 2026-06-20

### 新增

- 新增调度中心页面和 `/api/v1/admin-plus/scheduler/status`、`/api/v1/admin-plus/scheduler/run`，可按供应商状态生成 Chrome 插件采集任务。
- 新增 `schedule_key` 幂等键，防止同一供应商、任务类型和时间窗口重复创建插件任务。
- 新增插件任务结果摄取链路，插件完成任务后可把费率、余额、优惠、健康和账单结果写入 Admin Plus 运营数据。
- 新增最小 Chrome MV3 插件执行器，用于领取 Admin Plus 插件任务、按供应商后台页面执行采集并回传结果。
- 新增 `/api/v1/admin-plus/extension/tasks/:id/browser-credential`，插件仅能在持有有效任务租约时读取供应商浏览器登录凭据。
- 新增账号运行态页面和 `/api/v1/admin-plus/sub2api/account-runtime`，只读展示本地 Sub2API 账号状态、Redis 当前并发、等待队列和切换资格。
- 新增本地 Sub2API Redis 只读适配，支持读取 `concurrency:account:*`、`wait:account:*` 和 `temp_unsched:account:*`。

### 更新

- 扩展 Admin Plus API E2E 脚本，覆盖真实 HTTP、PostgreSQL、Redis 运行态、调度生成和插件结果摄取链路。
- 更新 README、代码结构文档、MVP 基线和 PRD，将调度、账号运行态和插件结果摄取进度收敛到当前事实。
- 更新版本号到 `0.6.0`，并同步 DockerHub 手动发布默认标签。

### 修复

- 插件任务调度改为持久化队列和幂等创建，避免手动或周期调度在同一窗口重复派发采集任务。
- 供应商浏览器登录凭据按任务租约读取，不在普通供应商或任务列表接口返回明文。
- 账号运行态只读 Redis 数据，不写入或清理 Sub2API 原生运行态 key，保持 Admin Plus 与 Sub2API 运行数据隔离。
