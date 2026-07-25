# Release Notes

## v0.43.4 - 2026-07-25

### 新增

- 新增 `aws-bedrock-anthropic-mask-2026-07` 签名家族，可将同时包含 Bedrock 字段族与 Anthropic 原生 metadata 的伪装链路继续归因为 AWS Bedrock。
- 报告新增 `score_adjustments`、`client_impact`、`impact_scope` 与 `failure_policy`，前后台均可展开查看渠道基线、逐项得分、判例和客户端影响。

### 改进

- 评分先按渠道选择基线；客户端能力失败按对应维度满扣，仅来源不透明且不影响客户端时扣 5 分，同一证据不重复处罚。标准 Bedrock 可得 100 分，Anthropic metadata mask 判例为 95 分。
- Token 用量异常检测默认关闭；未勾选时不发送额外 11 轮请求，验证项、维度、图表与 Detail 均隐藏。`not_run` 不展示，`not_applicable` 与 `unsupported_by_upstream` 继续展示。
- Admin Plus 与 `proxyaiweb` 的检测 Detail 默认折叠，支持逐项展开来源探针和得分；公开前端补齐中英日韩文案，品牌点击会重新载入首页并清空当前检测状态。

### 修复

- 修复混合字段 2/11 被旧互斥规则判为 unknown，导致 AWS Bedrock 上游未识别的问题。
- 修复最终 SSE 事件重新计算分项、把模型身份满扣后的 `tag_check=0` 覆盖回原始分数的问题。
- 修复未勾选 Token 审计时，后端 skipped validation 被动态追加回公开前端的问题。

### 测试

- 新增 Bedrock mask 归因、100/95 渠道评分、客户端维度满扣、重复处罚抑制和最终 SSE 分项一致性回归。
- 新增前后台 Token 审计隐藏、未执行维度过滤、Detail 折叠、评分调整、客户端影响和敏感字段脱敏回归。
- Admin Plus 前端 34 个测试文件、196 项测试通过；公开页 6 个测试文件、31 项测试通过。

### 发布

- 更新版本号到 `0.43.4`。
- GitHub Release 只发布 Linux 资产：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- tag 发布同步发布 DockerHub 与 GHCR 多架构镜像：`0.43.4`、`latest`、`0.43` 和 `0`。
- 裸机 systemd 部署通过 `sub2apiplus upgrade -v v0.43.4` 升级。
