# Release Notes

## v0.42.0 - 2026-07-25

### 新增

- 纯度检测固定输出 CCTest 兼容的 12 维检测矩阵，每一维独立返回状态、得分、来源探针、限制和脱敏 Detail。
- 新增结构化渠道归因与综合判定，区分原生渠道、AWS Bedrock、Google Vertex AI、透明中转、兼容渠道、渠道证据冲突和模型身份冲突。
- Claude 检测新增非流式与流式 thinking signature 结构探针，并使用校准后的脱敏指纹证据进行渠道归因。
- Admin Plus 账号检测弹窗新增结构化判定卡、12 维逐项 Detail、来源 Check Detail 和敏感字段过滤。

### 改进

- 网页检测、开发者 API 和后台账号检测的 Token 用量异常审计统一改为显式启用；未传或传 `false` 时不发送额外 11 轮请求。
- 未执行、不适用和上游不支持分别返回 `not_run`、`not_applicable`、`unsupported_by_upstream`，不再用笼统低分代替能力边界。
- Detail 总区块、每个检测维度和原始 Check 默认折叠，展开后同时显示逐项得分与证据细节。
- 指纹证据只参与渠道归因，不自动扣减协议兼容分；Token 审计关闭时显示“未检测/未评分”，不作为失败或扣分项。
- JSON、NDJSON 流、公共摘要、账号快照和 PDF 数据链路同步携带维度矩阵与结构化判定。
- Admin Plus 新增中英文判定、维度状态和 Detail 文案，保持现有 i18n 语言范围一致。

### 修复

- 修复后台账号检测未透传 Token 审计选择的问题，确保默认关闭和显式开启在同步、流式检测中行为一致。
- 修复不同上游来源共用单一结论的问题，避免将官方云渠道的能力限制误判为模型失败。
- 修复检测 Detail 可能暴露 API Key、Authorization、Cookie 或签名内容的问题。
- 升级 `golang.org/x/text`、`axios` 和 `postcss` 及其关联依赖，修复发布安全扫描识别的高危漏洞。

### 测试

- 增加 12 维矩阵、结构化判定、渠道冲突、模型冲突、Token 审计默认关闭和账号快照持久化测试。
- 增加 Admin Plus 弹窗默认关闭 Token 审计、折叠 Detail、逐项得分和敏感字段脱敏回归测试。
- 覆盖 OpenAI、Claude、Gemini 纯度服务、handler、路由、仓储以及两个前端的类型检查、单元测试和生产构建。

### 发布

- 更新版本号到 `0.42.0`。
- GitHub Release 只发布 Linux 资产：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- tag 发布同步发布 DockerHub 与 GHCR 多架构镜像：`0.42.0`、`latest`、`0.42` 和 `0`。
- 裸机 systemd 部署通过 `sub2apiplus upgrade -v v0.42.0` 升级，保留 `0.41.0` 回滚路径。
