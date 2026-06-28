# Release Notes

## v0.25.0 - 2026-06-29

### 新增

- 新增 Token audit 平台计费倍率探针，支持从账号配置或兼容站 `/usage` 标准费用与实际扣费增量推导 `billing_multiplier`。
- 新增 OpenAI Chat Completions usage 回退审计，在兼容站不支持 Responses 时保留 3 轮 usage 样本和 `chat_completions_audit_fallback` 诊断。
- 新增 Gemini GenerateContent 历史重放 Token audit，覆盖 `contents`、`systemInstruction`、`tools.functionDeclarations` 和 `usageMetadata` 字段观测。

### 改进

- 将 OpenAI Token audit 拆分为 `cache_probe` 与 `context_replay`，区分 cached_tokens 字段存在性、缓存命中、上下文重放链和 minimal retry。
- 将 Claude Token audit 改为 Messages 历史重放与 cache_control 形态，独立统计 cache creation/read 字段、缓存读写和 history replay 完整性。
- 统一 Admin Plus 前端和 PDF 的 Token audit 展示逻辑，优先展示 `overall_ratio`，单独展示平台计费倍率、失败轮次诊断和每轮延迟。
- 允许真实本机回环请求检测 `localhost` / `127.0.0.1` Base URL，公网入口仍默认阻断私网地址以降低 SSRF 风险。

### 修复

- 修正 OpenAI cached_tokens 为 0 但字段存在时被误判为字段缺失的问题。
- 修正无 usage 样本展示策略：图表不绘制 0 成本柱，明细表保留 0 值、状态码、错误分类和错误原因。
- 拆分 purity 大型测试文件，补充 billing multiplier、provider、stream/account、model identity、wrapper fingerprint、token audit 和 handler 回归测试。

### 发布

- 更新版本号到 `0.25.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub/Railway 镜像渠道不随常规发布自动执行；如需要，单独走“镜像渠道发布”流程。
