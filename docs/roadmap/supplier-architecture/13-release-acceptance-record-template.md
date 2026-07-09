# 13. P1/P2 发布验收记录模板

版本：v0.1.0
日期：2026-07-09
状态：A1-A12 人工验收留证模板

## 1. 使用方式

本模板用于记录 [10-release-acceptance-runbook.md](10-release-acceptance-runbook.md) 的 A1-A12 执行结果。

使用规则：

- 每次发布前复制一份本模板，填入发布 issue、飞书文档或内部验收记录。
- 不要填完整用户 API Key、完整第三方供应商 Key、完整请求 headers、完整 cookie、完整账号凭据或未脱敏生产请求体。
- 证据优先使用截图、脱敏响应摘要、`admin_plus_action_executions.id`、调度 `run_id/step_id`、补池 run id。
- A1-A9 任一失败时，发布结论必须是阻断。
- A10-A12 失败且本次发布包含对应能力时，发布结论必须是阻断；如果明确 scoped out，必须写灰度补验计划。

## 2. 验收批次

```text
验收日期：
验收环境：测试 / 预发 / 生产灰度
代码版本：commit SHA、tag 或镜像版本
镜像或二进制版本：
数据库迁移版本：
执行人：
回滚负责人：
上线后观察负责人：
关联发布单：
备份状态：已备份 / 不涉及 / 未完成
验收开始时间：
验收结束时间：
```

## 3. 自动校验记录

```text
go test -count=1 ./internal/adminplus/...：
go test -count=1 ./internal/handler/adminplus ./internal/server/routes ./cmd/server：
pnpm typecheck：
pnpm test:run：
git diff --check：
git status --short --ignored docs/aipromt AGENTS.md：
```

自动校验结论：

```text
结果：pass / fail
失败项：
处理结论：
```

## 4. A1-A12 单项记录

### A1 本地分组空池发现

```text
结果：pass / fail / skipped
证据：截图、动作建议 ID、接口响应摘要
问题链接：
处理结论：
```

### A2 补池 dry-run

```text
结果：pass / fail / skipped
证据：截图、dry-run 响应摘要、目标分组成员未变化证明
问题链接：
处理结论：
```

### A3 审批后真实补池

```text
结果：pass / fail / skipped
证据：recommendation id、execution id、补池 run id、前后容量摘要
问题链接：
处理结论：
```

### A4 补池影响追溯

```text
结果：pass / fail / skipped
证据：补池历史截图、脱敏受影响 Key 摘要、失败请求摘要
问题链接：
处理结论：
```

### A5 坏账号关调度建议

```text
结果：pass / fail / skipped
证据：动作建议 ID、通道失败信号、余额/配额不足未误触发证明
问题链接：
处理结论：
```

### A6 关调度执行与回滚

```text
结果：pass / fail / skipped
证据：执行 execution id、rollback execution id、前后快照摘要
问题链接：
处理结论：
```

### A7 原后台 drift 写前保护

```text
结果：pass / fail / skipped
证据：drift 截图、LOCAL_ACCOUNT_STATE_DRIFT_PENDING 响应、采纳或恢复记录
问题链接：
处理结论：
```

### A8 Key 配额开通计划

```text
结果：pass / fail / skipped
证据：开通计划截图、被阻塞分组、阻塞原因、allow_partial 行为
问题链接：
处理结论：
```

### A9 余额不足低倍率保护

```text
结果：pass / fail / skipped
证据：balance_blocked/recharge_required 截图、低倍率机会记录、未生成关调度证明
问题链接：
处理结论：
```

### A10 纯度过期受控复检

```text
结果：pass / fail / scoped out
证据：纯度过期账号截图、受控复检队列、复检 step/run id
问题链接：
处理结论：
```

### A11 调度来源追溯

```text
结果：pass / fail / scoped out
证据：scheduler_run_id、scheduler_step_id、execution 反跳截图或响应摘要
问题链接：
处理结论：
```

### A12 幂等 replay

```text
结果：pass / fail / scoped out
证据：Idempotency-Key、第一次响应摘要、replay 响应摘要、execution 数量证明
问题链接：
处理结论：
```

## 5. 脱敏检查

```text
完整用户 API Key：未出现 / 已清理
完整第三方供应商 Key：未出现 / 已清理
完整请求 headers：未出现 / 已清理
完整 cookie：未出现 / 已清理
完整账号凭据：未出现 / 已清理
未脱敏生产请求体：未出现 / 已清理
```

## 6. 最终结论

```text
自动校验：pass / fail
A1-A9：pass / fail
A10-A12：pass / fail / scoped out
非阻塞 skipped 项：
P3 记录项：未实施，不进入本轮验收
发布结论：放行 / 条件放行 / 阻断
发布后补验计划：
上线后观察负责人：
T+30 分钟观察计划：
T+2 小时观察计划：
T+24 小时观察计划：
```

## 7. 签字确认

```text
执行人：
复核人：
回滚负责人：
运营负责人：
最终确认时间：
```
