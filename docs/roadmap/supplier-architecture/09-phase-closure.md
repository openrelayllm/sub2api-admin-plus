# 09. P1/P2 阶段收口基线

版本：v0.1.0
日期：2026-07-09
状态：P1 主线收口，P2 第一阶段收口，P3 本轮不实施

## 1. 收口结论

当前阶段可以把 P1 和 P2 第一阶段关闭。后续继续推进时，不再把 P1.x/P2.x 增强项当作当前版本阻塞，P3 只记录边界且本轮不实施。

| 阶段 | 结论 | 关闭范围 |
|------|------|----------|
| P1 | 可以关闭 | 自动补池、坏账号关调度、Key 配额计划、余额门禁、统一候选评估、统一写回端口、动作执行历史、失败重试、成功回滚 |
| P2 | 可以关闭第一阶段 | 模型级候选、纯度检测联动、代理联动、通知矩阵、动作建议超时可视化、实测预算/冷却、接入验收步骤矩阵、利润缺口和建议价偏离 |
| P3 | 不实施 | 多 Sub2API 实例、跨实例容量、迁移冲突增强和外部事件适配不进入当前闭环 |

## 2. P1 关闭证据

| 验收点 | 当前证据 | 结论 |
|--------|----------|------|
| 写回统一端口 | `backend/internal/adminplus/app/sub2api/service.go` 定义 `Sub2APIRoutingPort`；补池和本地账号操作复用端口能力 | 通过 |
| 候选评估统一 | `backend/internal/adminplus/app/candidateeval/evaluator.go` 统一输出 `candidate_status/blocked_reason/check_source` | 通过 |
| Key 配额和批量开通计划 | `backend/internal/adminplus/app/supplierkeys/service.go` 使用供应商级、分组级配额和 Provider `ReadKeyCapacity` | 通过 |
| 余额门禁 | 候选评估和动作建议区分 `balance_blocked/recharge_required`，余额不足不触发渠道坏判断 | 通过 |
| 本地分组补池 | `backend/internal/adminplus/app/sub2api/routing_refill.go` 支持 dry-run、真实写回、冷却、确认窗口、失败候选抑制和 `model_scope` | 通过 |
| 坏账号关调度 | `local_account_schedule_disable` 动作复用本地账号运营 preview/apply，不自动关闭余额不足或配额不足账号 | 通过 |
| 动作执行历史 | `admin_plus_action_executions` 承载补池、关调度、手工本地账号操作、成本对账执行记录 | 通过 |
| 失败重试和成功回滚 | 动作建议执行历史支持 `routing_refill/local_account_schedule_disable` 的 retry 和 rollback | 通过 |

P1 不再阻塞的增强项：

- 真实最大 Key 上限自动读取：依赖第三方是否暴露稳定上限字段；当前 new-api/sub2api 已能读取 active Key 数，但 `LimitKnown=false` 时仍以运营配置为准。
- 账单明细自动定位和批量导入：属于财务运营增强，不影响补池、门禁和调度闭环。
- 非本地路由类动作的通用回滚：后续按 Provider Adapter 语义逐类扩展。

## 3. P2 第一阶段关闭证据

| 验收点 | 当前证据 | 结论 |
|--------|----------|------|
| 模型级候选 | `model_scope/model_match_status` 已进入候选评估和补池运行；明确不匹配输出 `model_scope_unsupported`，未知不阻断 | 通过 |
| 纯度检测联动 | 候选读模型复用最近 `run_purity_check` step，输出 `purity_failed/purity_risk/purity_stale`；前端支持复检深链和当前页受控复检队列 | 通过 |
| 代理联动 | 读取 Sub2API `accounts.proxy_id -> proxies`，明确 deleted/disabled/expired/error 代理阻断，unknown 或无代理不误阻断 | 通过 |
| 通知矩阵 | 动作建议按余额、Key 配额、路由容量、本地状态、通道失败、代理、纯度、成本和利润风险映射到通知中心 | 通过 |
| 超时未处理可视化 | `frontend/src/views/admin/operations/ActionRecommendationsView.vue` 基于 `created_at/status/severity` 派生严重 2 小时、警告 12 小时、普通 24 小时超时统计 | 通过 |
| 实测预算/冷却 | `channelchecks.Check` 支持每日 token 预算、单次估算 token 和同分组冷却；预算或冷却跳过不会写失败快照 | 通过 |
| 接入验收步骤矩阵 | Kanban 验收报告按连通性、模型列表、纯度、轻量调用、Usage、缓存、余额和并发汇总阻断点 | 通过 |
| 成本利润第一阶段 | 模型利润行展示目标毛利缺口和建议价相对市场中位价偏离 | 通过 |

P2 不再阻塞的增强项：

- 通知自动升级投递、值班分组、多渠道通知。
- Admin Plus 代理中心节点健康、出口失败、fallback 建议和本地账号绑定代理深度联动。
- 跨页或后台批量纯度复检、按模型预算归集。
- 完整财务汇总：成本、收入、毛利、异常对账和账单批量导入。
- 主动实测按模型、动作来源和原因做细粒度成本归集。

## 4. 当前版本发布前验收清单

发布当前阶段前，只需要验证核心闭环，不需要补齐 P2.x。

核心闭环按三个层次验收：

- P1 调度闭环：空池发现、补池 dry-run、真实补池、影响追溯。
- P1 安全闭环：坏账号关调度、回滚、原后台 drift 写前保护、Key 配额和余额门禁。
- P2 第一阶段可信性：纯度受控复检、调度来源追溯、幂等 replay。

### 4.1 可执行验收门禁

人工验收建议在测试环境或可回滚的运营窗口执行。每一项都应保存页面截图、请求响应摘要或执行记录 ID；不要用生产敏感 Key 明文作为验收材料。

逐项操作步骤、失败处置和放行记录模板见 [10-release-acceptance-runbook.md](10-release-acceptance-runbook.md)。本节只保留关闭基线和最小验收门禁。

P1 硬门禁：

**A1 本地分组空池发现**

- 入口/API：`/admin/scheduler`、`GET /api/v1/admin-plus/sub2api/groups`。
- 必须证明：空池或低容量可见，并能跳转到 `routing_refill` 动作建议。

**A2 补池 dry-run**

- 入口/API：`/admin/local-account-ops`、`/admin/scheduler`、`POST /api/v1/admin-plus/sub2api/routing/refill-local-group`。
- 必须证明：返回最低倍率候选、`availability_before` 和用户影响摘要，且不写入目标分组。

**A3 审批后真实补池**

- 入口/API：`/admin/actions`、`POST /api/v1/admin-plus/actions/recommendations/:id/execute`。
- 必须证明：审批后写回成功进入 `admin_plus_action_executions`，成功时目标分组可调度账号增加。

**A4 补池影响追溯**

- 入口/API：`/admin/scheduler/routing-refill-history`、`GET /api/v1/admin-plus/sub2api/routing/group-impact/api-keys`。
- 必须证明：可看到运行状态、候选、前后容量、脱敏受影响 Key 和失败摘要；敏感明细查询必须要求原因。

**A5 坏账号关调度建议**

- 入口/API：`/admin/actions`、`GET /api/v1/admin-plus/actions/recommendations`。
- 必须证明：通道失败账号生成 `local_account_schedule_disable`；余额不足、Key 配额不足、drift 不生成坏账号关调度建议。

**A6 关调度执行与回滚**

- 入口/API：`/admin/actions`、`POST /api/v1/admin-plus/actions/recommendations/:id/execute`、`POST /api/v1/admin-plus/actions/recommendations/:id/executions/:executionID/rollback`。
- 必须证明：执行记录包含前后快照；回滚新建 execution，并保留 `rollback_source_execution_id`。

**A7 原后台 drift 写前保护**

- 入口/API：`/admin/local-account-ops`、`POST /api/v1/admin-plus/sub2api/local-account-ops/sync-local-state`。
- 必须证明：原后台变更可见；写回被 `LOCAL_ACCOUNT_STATE_DRIFT_PENDING` 阻断；可采纳或恢复基线。

**A8 Key 配额开通计划**

- 入口/API：供应商分组弹窗、`POST /api/v1/admin-plus/suppliers/:id/keys/ensure-all-plan`。
- 必须证明：计划展示可创建、已覆盖、被阻塞分组和阻塞原因；未显式 `allow_partial` 时不静默部分创建。

**A9 余额不足低倍率保护**

- 入口/API：`/admin/actions`、`/admin/local-account-ops`、供应商详情弹窗。
- 必须证明：候选显示 `balance_blocked/recharge_required`；保留低倍率机会；不生成渠道坏或关调度建议。

P2 第一阶段可信性门禁：

**A10 纯度过期受控复检**

- 入口/API：`/admin/local-account-ops`、`/admin/actions`。
- 必须证明：复检由运营显式触发；批量复检只进入受控队列，不自动批量消耗 token。

**A11 调度来源追溯**

- 入口/API：`/admin/scheduler` 运行详情弹窗、`GET /api/v1/admin-plus/scheduler/runs/:id`、`/admin/actions`。
- 必须证明：execution 记录 `scheduler_run_id/scheduler_step_id`，执行历史可反跳回调度运行详情。

**A12 幂等 replay**

- 入口/API：本地账号运营 apply API、补池 apply API、动作建议执行 API。
- 必须证明：相同 `Idempotency-Key` 不新增重复 execution、不重复写回，并标记 `idempotency_replayed=true`。

### 4.2 代码级证据清单

本清单只证明入口、接口、服务链路和测试覆盖存在，不替代第 4.1 节的真实数据验收。发布前仍必须在测试、预发或生产灰度环境留存 A1-A12 的运行证据。为避免宽表在 Markdown 中难以阅读，本节按验收项分块记录。

**A1 本地分组空池发现**

- 代码级证据：`backend/internal/server/routes/admin_plus_surface_test.go` 覆盖 `GET /api/v1/admin-plus/sub2api/groups` 和 `/api/v1/admin-plus/actions/recommendations`；`backend/internal/adminplus/app/scheduler/service_test.go` 覆盖容量 watch 生成 `routing_refill` 动作；`frontend/src/router/adminPlusRoutes.ts` 和侧栏测试覆盖 `/admin/scheduler`、`/admin/actions`。
- 仍需人工确认：真实环境存在空池或低容量本地分组，并能从容量矩阵跳转到补池建议。

**A2 补池 dry-run**

- 代码级证据：`backend/internal/handler/adminplus/sub2api_handler.go` 暴露 `RefillLocalGroup`；`backend/internal/adminplus/app/sub2api/routing_refill_test.go` 覆盖 dry-run 选择最低倍率候选、模型匹配、冷却和确认窗口；`routing_refill.go` 返回 `availability_before`。
- 仍需人工确认：dry-run 页面或 API 响应确认没有写入目标本地分组。

**A3 审批后真实补池**

- 代码级证据：`backend/internal/handler/adminplus/action_handler.go` 和 `sub2api_handler.go` 承载动作执行；`backend/internal/adminplus/app/actions/service_test.go` 覆盖 `routing_refill` 外部执行记录；`backend/internal/handler/adminplus/sub2api_action_handler_test.go` 覆盖执行记录写入幂等键和调度来源。
- 仍需人工确认：审批后的真实补池在 `admin_plus_action_executions` 留下本次执行记录，并可看到补池后容量变化。

**A4 补池影响追溯**

- 代码级证据：`backend/internal/server/routes/adminplus.go` 暴露 `GET /routing/group-impact/api-keys`、`GET /routing/group-impact/failures` 和敏感明细 reason 接口；`backend/internal/adminplus/app/sub2api/sql_repository_test.go` 覆盖补池运行历史、失败请求脱敏和敏感字段读取；前端入口 `/admin/scheduler/routing-refill-history` 已在路由测试覆盖。
- 仍需人工确认：补池影响历史能关联到本次运行，并按原因查询敏感明细。

**A5 坏账号关调度建议**

- 代码级证据：`backend/internal/adminplus/app/actions/service_test.go` 覆盖 `local_account_schedule_disable_required`，并验证余额阻断候选走充值建议；`backend/internal/adminplus/app/candidateeval/evaluator.go` 区分余额、配额、纯度、代理和通道信号。
- 仍需人工确认：真实通道失败账号生成关调度建议，余额不足或配额不足账号不生成坏账号关调度建议。

**A6 关调度执行与回滚**

- 代码级证据：`backend/internal/handler/adminplus/sub2api_action_handler_test.go` 覆盖 `local_account_schedule_disable` 执行、失败重试和成功回滚；回滚 payload 保留 `rollback_source_execution_id`；前端 `ActionRecommendationsView.vue` 暴露 retry/rollback 操作。
- 仍需人工确认：真实执行和回滚各产生一条 execution，且本地账号调度状态符合预期。

**A7 原后台 drift 写前保护**

- 代码级证据：`backend/internal/adminplus/app/sub2api/sql_repository_test.go` 覆盖 pending drift 阻断写回、同步、采纳和恢复；`backend/internal/handler/adminplus/sub2api_handler.go` 暴露 sync/accept/restore；前端 `LocalAccountOpsView.vue` 展示 drift 弹窗和 `LOCAL_ACCOUNT_STATE_DRIFT_PENDING` 提示。
- 仍需人工确认：在 Sub2API 原后台手工改动后，Admin Plus 能检测差异且写前保护生效。

**A8 Key 配额开通计划**

- 代码级证据：`backend/internal/server/routes/admin_plus_surface_test.go` 覆盖 `POST /api/v1/admin-plus/suppliers/:id/keys/ensure-all-plan`；`backend/internal/adminplus/app/supplierkeys/service_test.go` 覆盖有限配额最低倍率优先、分组配额阻断、Provider active Key 读取、未知配额阻断和显式部分开通；`ReadKeyCapacity` 已在 new-api/sub2api adapter 测试覆盖分页读取。
- 仍需人工确认：真实供应商配额有限时，计划明确展示阻塞原因，未显式 `allow_partial` 时不静默部分创建。

**A9 余额不足低倍率保护**

- 代码级证据：`backend/internal/adminplus/app/candidateeval/evaluator.go` 输出 `balance_blocked/recharge_required`；`backend/internal/adminplus/app/actions/service_test.go` 覆盖余额不足生成充值建议而不是关调度；前端 `ActionRecommendationsView.vue` 和 `LocalAccountOpsView.vue` 展示低倍率余额机会与余额阻断标签。
- 仍需人工确认：低倍率但余额不足的账号仍保留为充值机会，不被标记为渠道坏。

**A10 纯度过期受控复检**

- 代码级证据：`backend/internal/adminplus/app/candidateeval/evaluator_test.go` 覆盖 `purity_stale`；`backend/internal/adminplus/app/actions/service_test.go` 覆盖 `candidate_purity_stale` 复检建议；`backend/internal/handler/adminplus/purity_handler.go` 和 `frontend/src/views/admin/operations/LocalAccountOpsView.vue` 支持账号纯度复检和当前页队列。
- 仍需人工确认：批量复检只逐个打开受控队列，不默认后台批量消耗 token。

**A11 调度来源追溯**

- 代码级证据：`backend/internal/server/routes/admin_plus_surface_test.go` 覆盖 `GET /api/v1/admin-plus/scheduler/runs/:id`；`frontend/src/views/admin/scheduler/SchedulerRunDetailDialog.vue` 把 run/step 来源带入 `/admin/actions`；`frontend/src/views/admin/operations/ActionRecommendationsView.vue` 执行时传递并反跳 `scheduler_run_id/scheduler_step_id`；后端 action execution 结构持久化来源字段。
- 仍需人工确认：从真实 run/step 进入并执行后，execution 可反跳回对应调度运行详情。

**A12 幂等 replay**

- 代码级证据：`backend/internal/handler/adminplus/idempotency_helper.go` 统一处理 `Idempotency-Key`；`backend/internal/handler/adminplus/sub2api_action_handler_test.go` 覆盖本地账号 apply 和动作执行 replay 标记；`backend/internal/adminplus/app/actions/sql_repository_test.go` 覆盖 `idempotency_replayed` 写回。
- 仍需人工确认：真实重放同一写动作不重复写本地分组/调度，也不新增重复 execution。

### 4.3 不纳入当前发布阻塞

以下项目可以继续排入 P1.x/P2.x 后续增强；P3 只记录为不实施边界。它们都不应阻塞当前 P1/P2 收口验收：

| 队列 | 不阻塞项 | 原因 |
|------|----------|------|
| P1.x | 真实最大 Key 上限自动读取 | new-api/sub2api 当前可读取 active Key 数；最大上限依赖具体供应商是否暴露稳定字段 |
| P1.x | 账单明细自动定位、重复账单冲突预检和批量导入 | 当前已有人工对账调整和明细修复第一阶段；自动定位属于财务运营增强 |
| P2.x | 通知自动升级、值班分组和多渠道投递 | 当前通知矩阵第一阶段已复用通知中心规则、投递、去重和静默窗口 |
| P2.x | 代理中心深度质量联动和 fallback 建议 | 当前候选评估已读取本地账号绑定代理状态，明确代理不可用会阻断 |
| P2.x | 容量矩阵行内今日请求、限流账号、错误账号和最低可补倍率 | 这些指标已在补池影响面板和动作建议信号中用于执行确认，矩阵行内展示是扫盘效率增强 |
| P2.x | 跨页/后台纯度复检和按模型预算归集 | 当前支持当前页模型/能力圈选和受控复检队列，避免默认自动消耗 token |
| P3 | 多 Sub2API 实例、外部事件、迁移冲突增强 | 当前决策明确不实施 P3；远程写回第一阶段只保留为单实例远程端口能力 |

### 4.4 验收记录模板

每次发布前验收应创建一份记录，最少包含验收环境、代码版本、数据库迁移版本、执行人、回滚负责人、A1-A12 结果、证据链接、问题链接和处理结论。

完整可复制模板统一维护在 [10-release-acceptance-runbook.md](10-release-acceptance-runbook.md#6-放行记录模板)。不要把生产密钥、完整用户 API Key、完整第三方 Key、请求体或 headers 放入记录。

### 4.5 放行与阻断规则

| 结论 | 条件 | 处理 |
|------|------|------|
| 放行 | A1-A12 全部 pass，且自动校验记录中的命令在最终代码基线通过 | 可以进入发布或灰度 |
| 条件放行 | 仅 P1.x/P2.x 非阻塞增强未完成，或 P3 记录项未实施，且已在验收记录写明跳过原因、影响范围和后续负责人 | 可以灰度；不得把 skipped 项宣传为已完成 |
| 阻断 | A1-A9 任一 fail，或出现余额不足被误判为渠道坏、Key 配额静默部分创建、drift 被覆盖、无审计写回、重复写回等核心闭环问题 | 停止发布，修复后重跑相关验收 |
| 阻断 | A10-A12 任一 fail 且影响纯度受控复检、调度来源追溯或幂等 replay 的可信性 | 停止发布，除非发布范围明确不包含相关功能且有回滚方案 |
| 阻断 | 验收证据包含生产敏感明文、完整 Key、请求体或 headers | 清理证据并重新留存脱敏材料 |

最小放行口径：

- A1-A9 是 P1 主线和运营闭环硬门禁。
- A10-A12 是 P2 第一阶段可信性门禁；如果跳过，必须说明当前发布不触发对应能力，并在灰度计划中保留补验时间。
- 自动校验命令必须基于最终待发布代码重新执行；不能复用改动前的旧结果。
- 发布或灰度后的观察、降级和回滚按 [11-post-release-ops-runbook.md](11-post-release-ops-runbook.md) 执行。
- 发布后若运营仍需要回 Sub2API 原后台完成主路径操作，应视为当前版本未达到“Admin Plus 日常主操作入口”的目标，需要回到 P1/P2 问题池处理。

当前证据状态：

- 本地自动校验已在 2026-07-09 当前工作树通过，记录见第 7.2 节。
- A1-A12 仍需要测试、预发或生产灰度环境的人工验收材料；没有这些材料时，不能把当前版本标记为“已发布验收通过”。
- 如果后续继续修改代码、迁移、前端入口、路由或发布脚本，必须重新执行第 7.1 节命令，并按 [10-release-acceptance-runbook.md](10-release-acceptance-runbook.md) 重跑受影响的人工验收项。

## 5. 后续 Backlog 边界

后续继续迭代时按以下边界拆分，避免把 P3 或大而全能力拉回当前闭环。

| 队列 | 内容 | 是否阻塞当前版本 |
|------|------|------------------|
| P1.x | 真实最大 Key 上限自动读取、账单明细自动定位、批量账单导入、非本地路由动作回滚 | 否 |
| P2.x | 通知升级、代理深度质量、跨页纯度复检、完整财务汇总、细粒度实测成本 | 否 |
| P3 | 多 Sub2API 实例、跨实例容量、迁移冲突增强、外部事件 | 不进入本轮实施和验收 |

## 6. 不再扩大当前阶段的约束

1. 不修改 `/Users/coso/Documents/dev/go/sub2api` upstream。
2. 不做请求热路径 hook patch。
3. 不把余额不足当作渠道坏。
4. 不把主动实测作为默认第一检查。
5. 不把 Chrome 插件作为最终业务事实源。
6. 不做没有 dry-run、审计和写前保护的批量写操作。

## 7. 自动校验记录

本记录只证明当前代码基线通过自动化校验，不替代第 4 节的真实运营场景验收。

### 7.1 发布前必须重跑命令

发布前必须在最终待发布代码基线重新执行以下命令。任一命令失败时，不得沿用历史通过记录放行。

| 顺序 | 命令 | 通过标准 |
|------|------|----------|
| 1 | `cd backend && go test -count=1 ./internal/adminplus/...` | Admin Plus 后端应用、适配器、候选评估、补池、动作、通知、供应商 Key、导入导出、纯度和调度相关包全部通过 |
| 2 | `cd backend && go test -count=1 ./internal/handler/adminplus ./internal/server/routes ./cmd/server` | Admin Plus HTTP handler、路由注册和 server 入口全部通过 |
| 3 | `cd frontend && pnpm typecheck` | 前端 Vue/TypeScript 类型检查通过 |
| 4 | `cd frontend && pnpm test:run` | 前端 Vitest 回归通过；若出现测试内预期 stderr 或 browserslist 提示，应确认最终结果仍为通过 |
| 5 | `git diff --check` | 最终待发布 diff 无行尾空格、冲突标记和 patch 格式问题 |
| 6 | `git status --short --ignored docs/aipromt AGENTS.md` | 确认本地 AI 资料和 `AGENTS.md` 仍按规则保持 ignored，不进入待提交范围 |

如果发布流程同时修改 GitHub Actions、Docker、GoReleaser、Railway 或部署脚本，还需要按发布 runbook 单独验证对应 workflow 配置；不得通过本节命令推断发布链路已经可用。

### 7.2 已执行自动校验记录

| 日期 | 命令 | 结果 | 覆盖范围 |
|------|------|------|----------|
| 2026-07-09 | `go test -count=1 ./internal/adminplus/...` | 通过 | Admin Plus 后端应用、适配器、候选评估、补池、动作、通知、供应商 Key、导入导出、纯度、调度等包 |
| 2026-07-09 | `go test -count=1 ./internal/handler/adminplus ./internal/server/routes ./cmd/server` | 通过 | Admin Plus HTTP handler、服务端路由注册和 server 入口 |
| 2026-07-09 | `pnpm typecheck` | 通过 | 前端 Vue/TypeScript 类型检查，覆盖动作建议、本地账号运营、供应商和调度相关类型引用 |
| 2026-07-09 | `pnpm test:run` | 通过 | 前端 Vitest 回归，32 个测试文件、193 个测试通过 |
| 2026-07-09 | `git diff --check` | 通过 | 当前工作树 diff 无行尾空格、冲突标记和 patch 格式问题 |
| 2026-07-09 | `git status --short --ignored docs/aipromt AGENTS.md` | 通过 | `AGENTS.md` 和 `docs/aipromt/` 保持 ignored，不进入待提交范围 |
| 2026-07-09 | `README/09/10/11/12/13 markdown relative link check` | 通过 | 收口入口、阶段关闭、发布前验收、上线后观察、发布就绪快照和验收记录模板文档的相对链接均可解析到本地文件 |
| 2026-07-09 | `A1-A12 evidence file existence check` | 通过 | A1-A12 代码级证据中列出的核心后端路由、服务、测试和前端入口文件均存在 |

## 8. 文档一致性记录

| 日期 | 文档 | 调整 |
|------|------|------|
| 2026-07-09 | `docs/roadmap/routing/README.md` | 从“自动执行器待实施”的旧口径更新为 P1 已落地口径；明确调度 worker、动作建议执行历史、失败重试和成功回滚已完成，外部事件、多实例和静默自动关调度不进入本轮 P1/P2 收口；分布式锁描述改为远程写回或未来恢复多实例时才需要 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/04-local-binding-routing.md` | 收敛本地写回边界：当前写回、自动补池和坏账号关调度已沿 `Sub2APIRoutingPort` 落地；P3 多实例本轮不实施 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/05-operations-visualization.md` | 明确 24 小时请求/错误/429/token 已在补池影响面板落地；容量矩阵行内今日请求、限流、错误和最低可补倍率归入 P2.x 可视化增强，不阻断当前收口 |
| 2026-07-09 | `docs/roadmap/kanban/README.md` | 从“真实路由执行仍在后续阶段”的旧口径更新为本地路由类 `routing_refill/local_account_schedule_disable` 已进入统一 action execution；线上切流和权重调整仍保持后续阶段；补充说明本文 P0/P1/P2/P3 是看板专项阶段名 |
| 2026-07-09 | `docs/roadmap/restructure/README.md` | 标记为历史重构计划，当前 P1/P2 收口状态改以 supplier architecture 和本文件为准；补充说明本文 P0/P1/P2/P3 是历史阶段名 |
| 2026-07-09 | `docs/roadmap/billing/README.md` | 补充说明账务 P1/P2/P3/P4 是账务专项阶段名，下游收入、利润对账和闭账调整不阻塞当前 supplier architecture 收口 |
| 2026-07-09 | `docs/roadmap/accounts/ASYNC_PROVISIONING.md` | 标记为账号开通异步治理专项，真实 E2E 待收口保留为专项风险，不再误读为 P1/P2 全局收口阻塞 |
| 2026-07-09 | `docs/roadmap/scheduler/README.md` | 标记调度底座 Redis Stream 唤醒和计划编辑向导为 scheduler 专项增强；明确 `local.sub2api.routing.capacity_watch` 已接入 worker，默认生成动作建议，开启自动补池后才真实写回空池分组 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/07-iteration-plan.md` | 更新日期并收敛 P3 口径：P3 只保留记录，不进入本轮实施和验收；未来若恢复多实例也必须沿 `Sub2APIRoutingPort` 扩展 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/08-database-design.md` | 更新日期并收敛后续写回边界：P1.x/P2.x 新增写回仍只能走 `Sub2APIRoutingPort`；P3 本轮不实施 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/README.md` | 增加“接手者先读”和 09/10/11 收口链路入口，明确 09 定义关闭基线，10 执行发布前验收，11 执行上线后观察、降级和回滚；补充当前证据状态，区分本地自动校验已通过和 A1-A12 人工验收待执行 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/09-phase-closure.md` | 补充当前证据状态：本地自动校验已通过，但没有真实环境 A1-A12 脱敏证据时不得标记为发布验收通过 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/10-release-acceptance-runbook.md` | 新增 A1-A12 发布前人工验收执行手册，承接本文件第 4 节的验收门禁、失败处置和放行记录模板；明确 P3 只是未实施记录项，不作为 P1.x/P2.x 增强宣传 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/11-post-release-ops-runbook.md` | 新增 P1/P2 上线后观察、风险降级、版本回滚和运营兜底手册，承接放行后的 T+30 分钟、T+2 小时和 T+24 小时观察 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/12-release-readiness-snapshot.md` | 新增当前发布就绪快照，汇总自动校验已完成、A1-A12 待人工验收、当前放行判定和下一步执行顺序 |
| 2026-07-09 | `docs/roadmap/supplier-architecture/13-release-acceptance-record-template.md` | 新增 A1-A12 发布验收记录模板，用于发布 issue、飞书文档或内部验收留证 |
