# Release Notes

## v0.43.0 - 2026-07-25

### 新增

- 新增渠道评分策略契约，按 Anthropic 原生、AWS Bedrock、Google Vertex、OpenAI 原生、Google AI Studio 和兼容协议选择独立评分基线。
- Claude thinking signature Detail 新增脱敏字段结构、指纹摘要、分类状态、渠道和置信度，不保存或返回原始签名。
- Admin Plus 与公开检测页同步展示评分渠道、评分维度上限和本渠道排除项。

### 改进

- 模型身份与渠道来源彻底解耦；new-api 支持的多来源能力列表不再被误用为当前 Claude 请求的模型身份冲突证据。
- AWS Bedrock 和 Google Vertex 使用官方云渠道基线；Bedrock 不支持 Anthropic 托管 WebSearch 的能力边界不会被误扣为协议失败，完整透明中转可得 100 分。
- Admin Plus 分值条使用后端返回的动态渠道权重，不再固定显示旧的 `20/30/30` 权重。
- 公开页浅色主题补齐主文字、次级文字、状态色和深色 surface 映射，提升 Detail 在桌面和移动端的对比度。
- 两端 Detail 与每项来源 Check 保持默认折叠，展开后显示逐项得分、渠道证据和脱敏技术细节。
- 仅在用户勾选 Token 用量异常检测后显示验证项、计量轴、能力项、分值和审计面板。

### 修复

- 修复未启用 Token 审计时后台仍显示“未启用”卡片、验证项和分值占位的问题。
- 修复公开页请求失败时丢失 `check_token_usage=false`、进而伪造 Token 审计失败项的问题。
- 修复浅色页面中灰色 Detail 文本、状态文字和图标对比度不足的问题。

### 测试

- 增加 Bedrock 透明中转 100 分、多来源网关不误判模型身份、动态评分权重和失败路径不显示 Token 审计的回归测试。
- 覆盖 purity 后端、handler、路由、仓储、Admin Plus 195 项前端测试和公开页 29 项测试，并完成两个前端的类型检查与生产构建。
- 使用 Playwright 在桌面与移动视口验证浅色 Detail、Bedrock 基线、折叠层级、Token 审计隐藏和横向溢出。

### 发布

- 更新版本号到 `0.43.0`。
- GitHub Release 只发布 Linux 资产：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- tag 发布同步发布 DockerHub 与 GHCR 多架构镜像：`0.43.0`、`latest`、`0.43` 和 `0`。
- 裸机 systemd 部署通过 `sub2apiplus upgrade -v v0.43.0` 升级，保留 `0.42.0` 回滚路径。
