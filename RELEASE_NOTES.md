# Release Notes

## v0.43.2 - 2026-07-25

### 新增

- 新增渠道端点注册表，参考 new-api 主流语言模型适配器识别 Anthropic、OpenAI、Google、AWS Bedrock、Azure OpenAI、阿里百炼、百度、智谱、腾讯混元、Moonshot、DeepSeek、Mistral、xAI、火山方舟和 Cloudflare Workers AI 等稳定官方渠道。
- 独立识别 OpenRouter、Dify、Coze、FastGPT、Submodel、AIProxy 等聚合或应用平台，以及 Kimi Coding、Z.AI Coding 和 OpenAI Codex Subscription；来源已识别不再等同于模型官方原生渠道。
- 新增渠道评分策略契约：先判定协议和上游渠道，再选择 Anthropic 原生、AWS Bedrock、Google Vertex、OpenAI 原生、Google AI Studio 或兼容协议基线。
- 检测 Detail 同时展示逐维度状态、得分、来源探针、渠道证据、评分基线、能力边界和脱敏技术字段，并保持默认折叠。

### 改进

- 模型身份、协议兼容、渠道来源和网关包装完全解耦；Qwen、DeepSeek 等模型经兼容协议时不再被错误判为厂商身份冲突。
- AWS Bedrock 和 Google Vertex 按官方云渠道能力评分，符合对应渠道规范时可得 100 分；Bedrock 不支持 Anthropic 托管 WebSearch 作为独立能力边界展示，不误扣其他 Messages 能力。
- Token 用量异常检测保持默认关闭；未勾选时不发送额外 11 轮请求，也不显示验证项、计量轴、能力项、分值、结论或 PDF 章节。
- Admin Plus 与公开检测页补齐中英文 Detail、探针、模式、限制和固定控件文案，不再在英文界面透传后端中文摘要或内部枚举。
- 公开页 Detail 使用独立高对比度浅色主题层，桌面、移动端和英文视口的小字号文本均满足可读性要求。

### 安全与准确性

- 公开 Detail 不返回内部去重哈希、完整 thinking signature、API Key 或认证字段。
- 自定义地址、Dify/Coze、任务平台、本地推理和单纯模型名不会被提升为官方渠道强证据，避免多来源中转场景误报。
- 检测器版本升级到 `channel-attribution/2026-07-25.3`。

### 测试

- 增加 Bedrock 透明中转、DeepSeek 官方端点兼容基线 100 分、多来源模型身份、签名脱敏、渠道端点和 Token 审计隐藏回归测试。
- Admin Plus 前端 34 个测试文件、196 项测试通过；公开页 6 个测试文件、30 项测试通过。
- Playwright 验证桌面、390px 移动端和英文视口无横向溢出、Detail 默认折叠、未启用 Token 审计不显示、英文不透传中文，且 Detail 低对比度项为 0。

### 发布

- 更新版本号到 `0.43.2`。
- GitHub Release 只发布 Linux 资产：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- tag 发布同步发布 DockerHub 与 GHCR 多架构镜像：`0.43.2`、`latest`、`0.43` 和 `0`。
- 裸机 systemd 部署通过 `sub2apiplus upgrade -v v0.43.2` 升级。
