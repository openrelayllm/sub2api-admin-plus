# 10. P1/P2 发布验收 Runbook

版本：v0.1.0
日期：2026-07-09
状态：P1/P2 第一阶段发布前人工验收手册

## 1. 使用范围

本 Runbook 用于执行 [09-phase-closure.md](09-phase-closure.md) 中的 A1-A12 发布前人工验收。

目标不是重新定义 P1/P2 范围，而是把当前关闭标准转成可执行步骤：

- P1 主线：必须通过 A1-A9。
- P2 第一阶段：必须通过 A10-A12，除非发布范围明确不包含对应能力，并有灰度补验计划。
- P3：本轮不实施，不进入本 Runbook。

本 Runbook 不替代自动化测试。发布前仍必须重跑 `09-phase-closure.md` 第 7 节列出的自动校验命令。

## 2. 验收前准备

### 2.1 环境要求

- 使用测试、预发或可回滚的生产灰度环境。
- 已完成数据库迁移，并记录迁移版本。
- 已登录 Admin Plus 注册用户账号；不要使用不存在的 root 账号假设。
- 当前环境可以访问本地 Sub2API 数据库和 Redis。
- Sub2API 原后台可作为应急备选入口，用于 A7 drift 验收。
- 至少准备一个可操作的测试本地分组、一个测试用户 API Key、一个测试供应商和一批测试本地账号。

### 2.2 数据要求

建议准备以下数据，避免在真实用户主流量上直接构造故障：

- 一个可临时置为空池或低容量的本地 Sub2API 分组。
- 一个启用状态的用户 API Key，绑定到该测试本地分组。
- 至少两个候选供应商账号：
  - 一个低倍率且余额正常。
  - 一个低倍率但余额不足，用于 A9。
- 一个可模拟通道失败或主动实测失败的本地账号，用于 A5/A6。
- 一个可在 Sub2API 原后台手工改分组或调度开关的本地账号，用于 A7。
- 一个 Key 配额有限或配额未知的供应商配置，用于 A8。
- 一个纯度检测缺失或超过 7 天的本地账号，用于 A10。
- 一个包含 `local.sub2api.routing.capacity_watch` 或相关本地路由动作的调度 run，用于 A11。

### 2.3 留证规则

每项验收至少保留一种证据：

- 页面截图。
- 脱敏后的请求响应摘要。
- `admin_plus_action_executions.id`。
- 调度 `run_id` 和 `step_id`。
- 补池 run id。

禁止留存：

- 完整用户 API Key。
- 完整第三方供应商 Key。
- 完整请求 headers。
- 完整账号凭据或 cookie。
- 未脱敏的生产请求体。

## 3. 执行顺序

建议按以下顺序执行，减少重复造数据：

1. 执行 A1-A4，先验证空池发现、补池 dry-run、真实补池和影响追溯。
2. 执行 A5-A6，验证坏账号关调度建议、执行、重试或回滚。
3. 执行 A7，验证原后台 drift 写前保护。
4. 执行 A8-A9，验证 Key 配额和余额不足低倍率保护。
5. 执行 A10-A12，验证 P2 第一阶段可信性门禁。
6. 汇总验收记录，按第 6 节判断放行、条件放行或阻断。

## 4. 操作卡片

### A1 本地分组空池发现

目标：确认 Admin Plus 能发现本地分组空池或低容量，并给出补池动作入口。

入口：

- `/admin/scheduler`
- `GET /api/v1/admin-plus/sub2api/groups`

步骤：

1. 准备一个测试本地分组，使其绑定启用用户 API Key，但可调度账号数为 0 或低于阈值。
2. 打开 `/admin/scheduler`。
3. 刷新调度中心容量矩阵或执行容量巡检。
4. 点击对应工作台动作，确认能跳转到 `/admin/actions?type=routing_refill&local_group_id=...`。

通过证据：

- 容量矩阵显示空池或低容量。
- 动作建议类型为 `routing_refill`。
- 留存本地分组 ID、动作建议 ID 或截图。

阻断条件：

- 空池分组未被发现。
- 无法跳转到补池建议。
- 分组仍有启用用户 API Key，但系统没有提示用户影响。

### A2 补池 dry-run

目标：确认补池预览能选择最低倍率候选，并且不会写入目标分组。

入口：

- `/admin/local-account-ops`
- `/admin/scheduler`
- `POST /api/v1/admin-plus/sub2api/routing/refill-local-group`

请求要点：

```json
{
  "local_group_id": 1001,
  "platform": "openai",
  "dry_run": true,
  "reason": "release_acceptance_a2"
}
```

步骤：

1. 选择 A1 的测试本地分组。
2. 执行补池预览，或调用 `dry_run=true` API。
3. 检查候选排序、最低有效倍率、`availability_before` 和用户影响摘要。
4. 重新检查目标本地分组成员，确认没有新增账号。

通过证据：

- 响应或页面展示最低倍率候选。
- 响应包含 `availability_before`。
- 目标分组成员没有变化。

阻断条件：

- dry-run 写入了目标分组。
- 未展示候选倍率或用户影响。
- 余额不足或配额不足账号被当作可直接补入候选。

### A3 审批后真实补池

目标：确认审批后的 `routing_refill` 能真实写回，并记录执行历史。

入口：

- `/admin/actions`
- `POST /api/v1/admin-plus/actions/recommendations/:id/execute`

步骤：

1. 打开 A1/A2 对应的 `routing_refill` 建议。
2. 将建议状态改为 `approved`。
3. 执行建议。
4. 查看执行历史和目标本地分组容量。

通过证据：

- `admin_plus_action_executions` 出现对应记录。
- 执行状态为 `succeeded`、`skipped` 或 `failed`，且有明确原因。
- 成功时目标分组可调度账号数增加。
- execution 包含 before/after 快照或补池结果摘要。

阻断条件：

- 未经审批即可执行真实补池。
- 成功补池没有 execution。
- 写回绕过 `Sub2APIRoutingPort` 导致无审计或无 `scheduler_outbox`。

### A4 补池影响追溯

目标：确认补池后能追溯容量变化、候选、受影响用户 Key 和失败请求摘要。

入口：

- `/admin/scheduler/routing-refill-history`
- `GET /api/v1/admin-plus/sub2api/routing/group-impact/api-keys`
- `GET /api/v1/admin-plus/sub2api/routing/group-impact/failures`

步骤：

1. 打开补池影响历史。
2. 定位 A3 的补池运行。
3. 查看候选、前后容量、受影响用户 Key 摘要。
4. 如需查看敏感失败明细，必须填写原因。

通过证据：

- 能看到补池 run 状态、候选和前后容量。
- 用户 Key 只显示脱敏摘要。
- 最近失败请求摘要可定位，但敏感明细需要原因。

阻断条件：

- 补池运行不可追溯。
- 受影响 Key 明文暴露。
- 查询敏感失败明细不需要原因。

### A5 坏账号关调度建议

目标：确认通道失败账号会生成关调度建议，同时余额不足或配额不足不会误判为坏账号。

入口：

- `/admin/actions`
- `GET /api/v1/admin-plus/actions/recommendations`

步骤：

1. 准备一个通道监控失败或主动实测失败且仍在调度的本地账号。
2. 生成或刷新动作建议。
3. 准备一个余额不足或 Key 配额不足的低倍率账号。
4. 再次生成或刷新动作建议。

通过证据：

- 通道失败账号生成 `local_account_schedule_disable`。
- 余额不足账号生成充值或余额机会，不生成坏账号关调度。
- Key 配额不足账号不生成坏账号关调度。

阻断条件：

- 余额不足被标记为渠道坏。
- Key 配额不足被标记为渠道坏。
- drift 未处理账号被自动生成静默关调度。

### A6 关调度执行与回滚

目标：确认关调度建议可以审批执行，并能安全回滚。

入口：

- `/admin/actions`
- `POST /api/v1/admin-plus/actions/recommendations/:id/execute`
- `POST /api/v1/admin-plus/actions/recommendations/:id/executions/:executionID/rollback`

步骤：

1. 打开 A5 生成的 `local_account_schedule_disable` 建议。
2. 审批并执行。
3. 确认本地账号 `schedulable=false`。
4. 对 succeeded execution 执行 rollback。
5. 确认本地账号 `schedulable=true` 或恢复到预期状态。

通过证据：

- 执行记录包含 before/after 快照。
- rollback 新建一条 execution。
- rollback payload 包含 `rollback_source_execution_id`。

阻断条件：

- 回滚覆盖了新的 drift。
- 回滚没有新 execution。
- 回滚后本地账号状态与预期不一致且无错误原因。

### A7 原后台 drift 写前保护

目标：确认 Sub2API 原后台应急操作不会被 Admin Plus 静默覆盖。

入口：

- `/admin/local-account-ops`
- Sub2API 原后台账号管理页。
- `POST /api/v1/admin-plus/sub2api/local-account-ops/sync-local-state`

步骤：

1. 在 Admin Plus 记录测试账号当前本地分组和调度状态。
2. 到 Sub2API 原后台修改该账号分组或调度开关。
3. 回到 Admin Plus 执行本地状态同步。
4. 尝试对该账号执行本地账号运营写动作。
5. 选择采纳原后台或恢复 Admin Plus 基线，确认状态变化。

通过证据：

- 页面显示 `原后台变更` 或 drift。
- 写回被 `LOCAL_ACCOUNT_STATE_DRIFT_PENDING` 阻断。
- 可采纳或恢复基线。

阻断条件：

- Admin Plus 自动覆盖原后台变更。
- drift 状态不可见。
- 采纳或恢复后没有审计记录。

### A8 Key 配额开通计划

目标：确认有限 Key 配额不会导致静默部分创建。

入口：

- 供应商分组弹窗。
- `POST /api/v1/admin-plus/suppliers/:id/keys/ensure-all-plan`

步骤：

1. 选择一个 Key 配额有限、未知或分组级受限的供应商。
2. 生成一键补齐所有分组 Key 的计划。
3. 检查可创建、已覆盖、被阻塞分组和阻塞原因。
4. 不传 `allow_partial` 时尝试执行计划。
5. 传入显式 `allow_partial=true` 后，再确认部分开通行为。

通过证据：

- 计划显示 `key_capacity_exhausted`、`key_capacity_unknown`、`group_key_capacity_exhausted` 或相关阻断原因。
- 未显式 `allow_partial` 时不会静默创建部分 Key。
- 显式部分开通时记录操作原因和结果。

阻断条件：

- 配额不足时仍静默创建部分 Key。
- 未展示哪些分组被阻塞。
- 真实最大上限未知时没有提示运营配置或确认。

### A9 余额不足低倍率保护

目标：确认余额不足不会让低倍率优质渠道被误判为不可用渠道。

入口：

- `/admin/actions`
- `/admin/local-account-ops`
- 供应商详情弹窗。

步骤：

1. 准备低倍率但余额不足的供应商或账号。
2. 刷新余额和候选状态。
3. 打开动作建议页和本地账号运营镜像。
4. 检查是否进入低倍率余额机会或充值建议。

通过证据：

- 候选状态显示 `balance_blocked` 或 `recharge_required`。
- 低倍率机会仍保留。
- 不生成 `local_account_schedule_disable`。

阻断条件：

- 余额不足被当作渠道坏。
- 余额不足账号被自动关调度。
- 页面无法区分余额不足与通道失败。

### A10 纯度过期受控复检

目标：确认纯度过期账号只能由运营显式触发复检，不会后台批量消耗 token。

入口：

- `/admin/local-account-ops`
- `/admin/actions`

步骤：

1. 准备纯度快照缺失或超过 7 天的账号。
2. 在本地账号运营页按当前页模型或能力标签圈选账号。
3. 启动批量复检队列。
4. 确认队列逐个打开复检弹窗。
5. 完成一个复检后查看调度 step 快照是否刷新。

通过证据：

- 页面显示纯度过期或 `purity_stale`。
- 批量复检队列需要运营显式触发。
- 不存在后台静默批量 token 消耗。
- 复检成功后候选评估读取最新 step 快照。

阻断条件：

- 打开页面即自动大批量复检。
- 复检无法限制在当前页模型或能力范围。
- 复检成功后候选状态不刷新。

### A11 调度来源追溯

目标：确认从调度 run/step 进入动作建议并执行后，执行历史能反跳回来源。

入口：

- `/admin/scheduler`
- `GET /api/v1/admin-plus/scheduler/runs/:id`
- `/admin/actions`

步骤：

1. 打开一个包含本地路由容量巡检或关调度动作的调度 run。
2. 在 run/step 详情中进入补池或关调度动作建议。
3. 审批并执行。
4. 查看 execution 中的 `scheduler_run_id` 和 `scheduler_step_id`。
5. 点击执行历史中的来源链接回到调度运行详情。

通过证据：

- action execution 保存 `scheduler_run_id`。
- 有 step 来源时保存 `scheduler_step_id`。
- 前端可反跳回 `/admin/scheduler?run_id=...&step_id=...` 或对应运行详情。

阻断条件：

- 调度来源丢失。
- 执行历史无法回到 run/step。
- 来源字段被写入错误 run。

### A12 幂等 replay

目标：确认重复点击或调度重试不会重复写回。

入口：

- 本地账号运营 apply API。
- 补池 apply API。
- 动作建议执行 API。

步骤：

1. 选择一个可回滚的本地账号写动作或补池动作。
2. 使用固定 `Idempotency-Key` 执行一次。
3. 使用完全相同 payload 和相同 `Idempotency-Key` 再执行一次。
4. 查询 execution 和本地分组/调度状态。

通过证据：

- 第二次响应带 replay 标记，或 execution 标记 `idempotency_replayed=true`。
- 不新增重复 execution。
- 不重复写本地分组或调度状态。

阻断条件：

- 相同幂等键导致重复写回。
- 相同幂等键新增重复 execution。
- replay 后无法追溯原执行记录。

## 5. 失败处置

### 5.1 必须阻断发布

出现以下任一情况必须停止发布：

- A1-A9 任一 fail。
- A10-A12 fail 且本次发布包含对应能力。
- 余额不足被误判为渠道坏。
- Key 配额不足时静默部分创建。
- drift 被自动覆盖。
- 真实写回没有审计记录。
- 相同幂等键重复写回。
- 验收证据包含未脱敏 Key、headers、cookie 或生产请求体。

### 5.2 可以条件放行

仅在以下情况可以条件放行：

- 失败项属于 P1.x/P2.x 非阻塞增强，或 P3 记录项未实施。
- 验收记录明确跳过原因、影响范围、负责人和补验时间。
- 发布说明不得把 skipped 项描述为已完成。

### 5.3 修复后重跑范围

- A1-A4 任一失败：重跑 A1-A4，并补跑 A11/A12。
- A5-A6 任一失败：重跑 A5-A6，并补跑 A7/A12。
- A7 失败：重跑 A7，并抽样重跑 A3/A6。
- A8 失败：重跑 A8，并检查供应商详情页配额提示。
- A9 失败：重跑 A5/A9，确认坏账号和余额不足不混淆。
- A10 失败：重跑 A10，并确认主动实测预算/冷却没有被绕过。
- A11 失败：重跑 A3/A6/A11。
- A12 失败：重跑所有发生写回的 A3/A6/A7/A8/A12。

## 6. 放行记录模板

可直接复制填写的独立模板见 [13-release-acceptance-record-template.md](13-release-acceptance-record-template.md)。本节保留最小字段，便于在 runbook 内快速查看。

验收批次：

```text
验收日期：
验收环境：
代码版本：
镜像或二进制版本：
数据库迁移版本：
执行人：
回滚负责人：
关联发布单：
备份状态：
验收开始时间：
验收结束时间：
```

单项记录：

```text
A1:
结果：
证据：
问题链接：
处理结论：

A2:
结果：
证据：
问题链接：
处理结论：

A3:
结果：
证据：
问题链接：
处理结论：

A4:
结果：
证据：
问题链接：
处理结论：

A5:
结果：
证据：
问题链接：
处理结论：

A6:
结果：
证据：
问题链接：
处理结论：

A7:
结果：
证据：
问题链接：
处理结论：

A8:
结果：
证据：
问题链接：
处理结论：

A9:
结果：
证据：
问题链接：
处理结论：

A10:
结果：
证据：
问题链接：
处理结论：

A11:
结果：
证据：
问题链接：
处理结论：

A12:
结果：
证据：
问题链接：
处理结论：
```

最终结论：

```text
自动校验：pass / fail
A1-A9：pass / fail
A10-A12：pass / fail / scoped out
非阻塞 skipped 项：
P3 记录项：未实施，不进入本轮验收
发布结论：放行 / 条件放行 / 阻断
发布后补验计划：
上线后观察负责人：
```

## 7. 完成定义

只有同时满足以下条件，才能把当前版本按 P1/P2 第一阶段收口发布：

1. `09-phase-closure.md` 第 7 节自动校验命令在最终代码基线通过。
2. A1-A9 全部 pass。
3. A10-A12 全部 pass，或明确 scoped out 且有灰度补验计划。
4. 所有证据已脱敏。
5. 没有未处理的阻断项。
6. 发布说明只描述已完成能力，不把 P1.x/P2.x skipped 项或 P3 记录项宣传为已完成。
7. 已指定上线后观察负责人，并按 [11-post-release-ops-runbook.md](11-post-release-ops-runbook.md) 准备 T+30 分钟、T+2 小时和 T+24 小时观察记录。
