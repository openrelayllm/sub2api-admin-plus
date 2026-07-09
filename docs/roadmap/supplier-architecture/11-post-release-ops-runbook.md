# 11. P1/P2 上线后观察与回滚 Runbook

版本：v0.1.0
日期：2026-07-09
状态：P1/P2 第一阶段发布后运营观察、风险处置和回滚手册

## 1. 使用范围

本 Runbook 用于执行 [10-release-acceptance-runbook.md](10-release-acceptance-runbook.md) 通过后的上线观察。

目标：

- 确认 Admin Plus 已成为日常主操作入口。
- 确认补池、关调度、drift 保护、Key 配额和余额门禁在线上真实流量下稳定。
- 在发现核心闭环问题时，能快速停用自动写动作、回滚应用版本或恢复运营兜底。

不覆盖：

- P3 多 Sub2API 实例能力。
- 修改 `/Users/coso/Documents/dev/go/sub2api` upstream。
- 通过生产数据库批量改数据来绕过 Admin Plus 写前保护。

## 2. 上线前确认

上线前必须完成：

1. [09-phase-closure.md](09-phase-closure.md) 第 7 节自动校验命令全部通过。
2. [10-release-acceptance-runbook.md](10-release-acceptance-runbook.md) A1-A12 完成验收，或明确 scoped out 并记录灰度补验计划。
3. 发布包、镜像、数据库迁移版本和回滚版本已记录。
4. 回滚负责人、运营负责人、值守窗口已确认。
5. 当前环境登录使用 Admin Plus 注册用户账号，不依赖 root 账号。
6. 已确认 Sub2API 原后台可作为应急备选入口。
7. 已确认关键操作不会把生产密钥、完整用户 API Key、完整第三方 Key、headers 或 cookie 贴入记录。

## 3. 灰度策略

### 3.1 推荐灰度顺序

1. 只读观察：
   - 打开本地账号运营、调度中心、动作建议、供应商详情。
   - 不启用自动补池。
   - 不执行批量写动作。
2. 人工写回：
   - 只执行少量 `routing_refill` 和 `local_account_schedule_disable`。
   - 每个动作都保留 execution id。
   - 验证可回滚。
3. 小范围自动化：
   - 只对测试本地分组或低风险本地分组启用自动补池。
   - 设置较长冷却窗口和确认窗口。
   - 观察 2 小时后再扩大范围。
4. 全量日常使用：
   - Admin Plus 作为主操作入口。
   - Sub2API 原后台仅作为应急备选。

### 3.2 上线后观察窗口

- T+0 到 T+30 分钟：只看核心错误、写回审计、drift 阻断和用户侧请求失败。
- T+30 分钟到 T+2 小时：观察补池建议、关调度建议、余额机会和 Key 配额阻断。
- T+2 小时到 T+24 小时：观察自动化是否减少人工切换，是否出现重复建议、重复写回或误关调度。
- T+24 小时后：进入日常运营节奏，仍按本 Runbook 的阻断项处理严重问题。

## 4. 观察面板

### 4.1 本地账号运营

入口：

- `/admin/local-account-ops`

观察点：

- `candidate_status` 是否稳定区分 available、balance_blocked、capacity_blocked、blocked。
- `blocked_reason` 是否正确展示 `recharge_required`、`LOCAL_ACCOUNT_STATE_DRIFT_PENDING`、`purity_stale`、`proxy_*` 等原因。
- 本地账号是否能按供应商、第三方分组、有效倍率、本地分组和调度状态筛选。
- drift 数量是否在运营可处理范围内。

异常信号：

- 余额不足账号被当成渠道坏。
- drift 账号仍被自动写回。
- 低倍率账号大量变成未知且没有原因。
- 运营仍必须回 Sub2API 原后台完成主路径切换。

### 4.2 调度中心

入口：

- `/admin/scheduler`
- `/admin/scheduler/routing-refill-history`

观察点：

- 空池和低容量分组是否能被发现。
- `local.sub2api.routing.capacity_watch` 是否生成合理动作建议。
- 自动补池是否只在启用策略后执行。
- 补池 run 是否记录前后容量、候选和跳过原因。

异常信号：

- 容量矩阵看不到明显空池。
- 自动补池越过确认窗口或冷却窗口。
- 补池 run 没有影响记录。
- 同一本地分组短时间内重复补同一账号。

### 4.3 智能动作

入口：

- `/admin/actions`

观察点：

- `routing_refill`、`local_account_schedule_disable`、`recharge_supplier`、`review_credential` 的数量和原因是否合理。
- 严重、警告、普通建议的超时统计是否能反映运营待办。
- execution 是否包含前后快照、幂等指纹、调度来源和回滚来源。

异常信号：

- 失败建议无法重试。
- 成功执行无法回滚。
- 同一 `Idempotency-Key` 产生重复 execution。
- 关调度建议来自余额不足或 Key 配额不足，而不是通道失败或主动实测失败。

### 4.4 供应商管理

入口：

- `/admin/suppliers`
- 供应商详情弹窗。
- 供应商分组弹窗。

观察点：

- 第三方分组、倍率、Key、余额、通道状态、本地绑定能被聚合展示。
- Key 配额有限或未知时，开通计划能提示阻断原因。
- 余额不足低倍率供应商保留为充值机会。

异常信号：

- Key 配额不足时静默部分创建。
- 供应商余额不足被标为供应商不可用。
- 第三方 Key 明文出现在页面、日志或验收记录。

### 4.5 操作审计

入口：

- `/admin/action-audits`
- `/admin/actions` execution 历史。

观察点：

- 本地账号写动作、补池、关调度、drift 同步、采纳、恢复都有审计。
- 失败、阻断、回滚能按账号、供应商、分组或 reason 定位。
- 调度 run/step 来源可反跳。

异常信号：

- 真实写回没有审计记录。
- 无法定位操作者或原因。
- 回滚记录覆盖旧执行记录，而不是新建 execution。

## 5. 关键指标

### 5.1 T+2 小时必须检查

- 新增 `routing_refill` execution 数。
- 新增 `local_account_schedule_disable` execution 数。
- failed execution 数和失败原因。
- `idempotency_replayed=true` 数量是否异常增加。
- pending drift 数量。
- 余额不足低倍率机会数量。
- Key 配额阻断数量。
- 主动实测次数和估算 token 消耗。
- 用户侧 429、5xx、空池相关失败是否上升。

### 5.2 T+24 小时必须检查

- 仍需通过 Sub2API 原后台完成的主路径操作次数。
- 补池后再次失败的账号数量。
- 被回滚的补池或关调度 execution 数量。
- 充值后恢复的低倍率供应商数量。
- 纯度过期复检队列处理量。
- P1.x/P2.x skipped 项是否仍在影响日常运营。

## 6. 回滚和降级

### 6.1 先降级功能，再回滚版本

发现异常时按以下顺序处理：

1. 停止扩大灰度范围。
2. 关闭自动补池，仅保留人工审批执行。
3. 暂停批量本地账号写动作。
4. 保留本地账号运营镜像、供应商详情和动作审计只读能力。
5. 如果只读能力也误导运营，再回滚应用版本。

### 6.2 必须立即降级

出现以下任一情况，立即关闭自动写动作：

- 自动补池重复写同一账号或同一分组。
- `local_account_schedule_disable` 因余额不足或配额不足误触发。
- drift 被覆盖。
- 相同幂等键重复写回。
- Key 配额不足时静默部分创建。
- 主动实测消耗超出预算或绕过冷却。

### 6.3 必须回滚版本

出现以下任一情况，应用版本必须回滚或停止发布：

- Admin Plus 无法登录或核心页面不可用。
- 本地账号运营页面显示的数据与 Sub2API 当前状态系统性不一致。
- 写回链路绕过 `Sub2APIRoutingPort`、无审计或无 `scheduler_outbox`。
- 用户侧请求失败明显上升，且无法通过关闭自动写动作恢复。
- 回滚按钮或 rollback API 不能恢复 P1 主路径动作。

### 6.4 回滚方式

按部署方式选择：

- systemd 二进制部署：使用已记录的 GitHub Release 版本回滚，例如 `sub2apiplus rollback vX.Y.Z`。
- Docker Compose 部署：把镜像 tag 固定回上一个已验证版本，再重启服务。
- Railway 部署：回滚到上一个健康 deployment，或重新部署上一版本镜像。

注意：

- 应用版本回滚不等于数据回滚。
- 不要直接删除 `admin_plus_action_executions`、补池运行历史或 drift 事件。
- 如果已发生错误写回，优先使用动作执行历史里的 rollback 或本地账号运营 apply 恢复。
- 数据库结构变更必须按对应 migration/runbook 判断，不要临时手写破坏性 SQL。

## 7. 运营兜底

### 7.1 Admin Plus 仍可用时

优先使用：

- `/admin/local-account-ops` 手工开启/关闭调度、加入/移出本地分组。
- `/admin/actions` retry/rollback。
- `/admin/action-audits` 定位动作原因。
- 供应商详情确认倍率、余额、Key 配额和本地绑定。

### 7.2 Admin Plus 写回不可用时

允许临时回到 Sub2API 原后台：

- 手工切换本地账号分组。
- 手工开关调度。
- 手工确认用户 API Key 绑定分组。

恢复后必须：

1. 回到 Admin Plus 执行本地状态同步。
2. 查看 drift。
3. 采纳原后台变更或恢复 Admin Plus 基线。
4. 记录应急操作原因和处理人。

### 7.3 第三方供应商后台兜底

只用于：

- 查看真实余额。
- 充值。
- 查看 Key 上限或后台限制。
- 验证第三方分组倍率。

不要把第三方后台作为最终事实源直接覆盖 Admin Plus 投影。恢复后应通过 Provider Adapter 或 Admin Plus 同步入口刷新事实。

## 8. 发布后记录模板

```text
上线版本：
上线时间：
观察窗口：
执行人：
回滚负责人：

T+30 分钟结论：
T+2 小时结论：
T+24 小时结论：

新增 routing_refill executions：
新增 local_account_schedule_disable executions：
failed executions：
rollback executions：
idempotency_replayed 次数：
pending drift 数量：
Key 配额阻断数量：
余额不足低倍率机会数量：
主动实测次数/估算 token：
用户侧 429/5xx 是否异常：

是否触发降级：
是否触发版本回滚：
是否仍需使用 Sub2API 原后台主路径：
后续 P1.x/P2.x 问题：
```

## 9. 完成定义

上线后满足以下条件，才算 P1/P2 第一阶段进入稳定运营：

1. T+24 小时内没有触发必须回滚版本的条件。
2. 自动写动作没有绕过审批、冷却、确认窗口、drift 保护或幂等保护。
3. 余额不足、Key 配额不足、通道失败三类状态没有混淆。
4. 运营主路径可以在 Admin Plus 完成，Sub2API 原后台只作为应急备选。
5. 所有真实写回都有 action execution、业务日志或可追溯审计。
6. P1.x/P2.x skipped 项已经进入后续 backlog，不再伪装成当前版本已完成能力。
