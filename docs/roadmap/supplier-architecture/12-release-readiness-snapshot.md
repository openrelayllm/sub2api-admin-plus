# 12. P1/P2 发布就绪快照

版本：v0.1.0
日期：2026-07-09
状态：代码、文档和本地自动校验已收口；真实环境 A1-A12 人工验收待执行

## 1. 结论

当前版本可以进入发布前人工验收阶段，但不能直接标记为“发布验收通过”。

- P1 主线：代码、文档和自动校验层面可以关闭。
- P2 第一阶段：代码、文档和自动校验层面可以关闭。
- P3：本轮不实施，不进入当前发布验收。
- 发布放行：必须按 [10-release-acceptance-runbook.md](10-release-acceptance-runbook.md) 完成 A1-A12，并留存脱敏证据。

## 2. 已完成证据

自动校验已在当前工作树通过，记录来源见 [09-phase-closure.md](09-phase-closure.md#72-已执行自动校验记录)。

- 后端 Admin Plus 全包测试通过。
- 后端 Admin Plus handler、routes、server 入口测试通过。
- 前端 Vue/TypeScript 类型检查通过。
- 前端 Vitest 回归通过，32 个测试文件、193 个测试通过。
- `git diff --check` 通过。
- `AGENTS.md` 和 `docs/aipromt/` 保持 ignored。
- README、09、10、11、12、13 的相对链接检查通过。
- A1-A12 代码级证据中列出的核心文件存在。

## 3. 未完成证据

以下证据必须在测试、预发或生产灰度环境补齐：

- A1-A9：P1 主线和运营闭环人工验收。
- A10-A12：P2 第一阶段可信性人工验收。
- 发布后观察：T+30 分钟、T+2 小时、T+24 小时记录。

没有这些证据时，只能说“本地自动校验已通过”，不能说“发布验收已通过”。

## 4. 当前放行判定

**当前判定：待人工验收。**

理由：

- 自动校验已通过。
- 关闭证据已经在 [09-phase-closure.md](09-phase-closure.md) 固化。
- 发布前操作步骤已经在 [10-release-acceptance-runbook.md](10-release-acceptance-runbook.md) 固化。
- 上线后观察和回滚已经在 [11-post-release-ops-runbook.md](11-post-release-ops-runbook.md) 固化。
- A1-A12 真实环境验收材料尚未留存。

## 5. 下一步执行顺序

1. 在最终待发布代码基线重跑 [09-phase-closure.md](09-phase-closure.md#71-发布前必须重跑命令) 的自动校验命令。
2. 按 [10-release-acceptance-runbook.md](10-release-acceptance-runbook.md) 准备测试分组、测试用户 API Key、测试供应商和测试本地账号。
3. 依次执行 A1-A12。
4. 把截图、脱敏响应摘要、execution id、run id 或补池 run id 填入 [13-release-acceptance-record-template.md](13-release-acceptance-record-template.md)。
5. 如果 A1-A9 任一失败，停止发布。
6. 如果 A10-A12 失败且本次发布包含对应能力，停止发布或明确 scoped out 和灰度补验计划。
7. 发布后按 [11-post-release-ops-runbook.md](11-post-release-ops-runbook.md) 记录 T+30 分钟、T+2 小时和 T+24 小时观察结果。

## 6. 不应继续扩大的范围

以下内容不进入当前发布验收：

- 多 Sub2API 实例。
- 跨实例容量。
- 迁移冲突增强。
- 外部事件适配。
- 通知升级、值班分组、多渠道通知。
- 代理中心深度质量、出口失败和 fallback 建议。
- 跨页或后台纯度复检。
- 完整财务汇总和细粒度实测成本归集。

这些内容只能作为 P1.x/P2.x 后续增强或 P3 不实施记录项，不应改变当前 P1/P2 第一阶段关闭标准。
