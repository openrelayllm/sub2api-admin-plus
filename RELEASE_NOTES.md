# Release Notes

## v0.24.0 - 2026-06-28

### 新增

- 新增 Gemini API Key 原生纯度检测流，支持模型列表、GenerateContent、流式、多模态和报告评分。
- 新增模型身份验证与包装指纹验证，报告输出请求模型、响应模型、疑似上游厂商和 wrapper signals。
- 新增纯度检测 PDF 导出，包含摘要、验证项、模型身份证据、Token audit 和探针明细。

### 改进

- 将 OpenAI、Claude、Gemini 探针、评分、报告、HTTP、URL、事件和渠道指纹拆分为更小职责模块。
- 增强 Claude 官方负向探针，覆盖 thinking signature、thinking budget 和 cache_control 约束。
- 增强 OpenAI Responses 正向探针、`openai-model` 响应头、`reasoning_tokens` 和 Token audit 失败诊断。
- 增强 CLIProxyAPI、new-api、sub2api、Antigravity、Bedrock、国产兼容模型等渠道 detector 与样本回归。

### 修复

- 修正 Token audit warning 对官方分的影响，跳过审计不再错误扣分。
- 修正无 usage 样本在前端图表和表格中的展示，保留失败原因但不显示 0 数据柱。
- 补充架构边界、样本校准、Gemini、模型身份和包装指纹回归测试。

### 发布

- 更新版本号到 `0.24.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub/Railway 镜像渠道不随常规发布自动执行；如需要，单独走“镜像渠道发布”流程。
