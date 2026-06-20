# Release Notes

## v0.5.0 - 2026-06-20

### 新增

- 新增本地 Sub2API 只读数据适配，支持读取真实 `accounts` 和 `usage_logs`。
- 新增本地用量页面，按账号和模型查看请求数、Token、收入、原始成本与延迟。
- 新增 `/api/v1/admin-plus/sub2api/usage-lines` 和 `/api/v1/admin-plus/sub2api/usage-summary`。
- 新增 Admin Plus API E2E 脚本，覆盖真实 HTTP、PostgreSQL、供应商绑定、本地用量、账单对账与动作建议链路。

### 更新

- 供应商账号绑定改为通过只读 Sub2API 数据源校验本地账号，并保存账号快照信息。
- 账单对账页面接入本地用量明细，减少静态或手工数据依赖。
- 更新 README、代码结构文档、MVP 基线和 PRD，实现进度收敛到当前事实。
- 更新版本号到 `0.5.0`，并同步 DockerHub 手动发布默认标签。

### 修复

- 去除 `admin_plus_supplier_accounts.local_sub2api_account_id` 对本地 `accounts` 表的跨库外键依赖，保持 Admin Plus 自有库与 Sub2API 生产库隔离。
