# Release Notes

## v0.26.0 - 2026-06-29

### 新增

- 新增 Gemini 可用模型回退选择，目标模型不可调度时可从 `/models` 返回的同协议模型中选择探针模型，并记录 `requested_model`、`probe_model` 和回退原因。
- 新增 Gemini Token audit minimal retry，遇到兼容网关拒绝历史重放请求时自动降级为最小 GenerateContent 请求并保留诊断。
- 新增 Gemini `cacheTokensDetails` 聚合支持，补齐部分网关只返回明细数组、不返回 `cachedContentTokenCount` 的缓存 token 统计。
- 新增平台计费倍率审计后的 `/v1/usage` 多次采样与快照比值回退，提升扣费延迟场景下的倍率确认能力。

### 改进

- 拆分 Gemini GenerateContent 文本探针和工具调用探针，避免工具能力异常影响基础协议可用性判断。
- Gemini 模型身份检查支持同协议探针回退通过，未发现跨厂商或降级伪装证据时不再误报失败。
- 透明中转或兼容网关在无模型、协议、签名、usage/cache 混淆证据时改为通过状态。
- 前端和 PDF 将“平台计费倍率”与“Usage 比值”分开展示，Gemini 缓存字段缺失或未命中时以 0 展示并附带说明。
- 增强 provider 归一化，支持 Gemini/Google/Vertex/Antigravity 等别名进入同一展示和审计口径。

### 修复

- 修正 Gemini `cachedContentTokenCount: 0` 被误判为字段缺失的问题。
- 修正兼容网关工具调用或多模态探针失败时误伤基础 GenerateContent 检测的问题。
- 修正 Sub2API/summary 多种 usage JSON 形态无法推导平台倍率的问题。
- 修正按 validation 状态粗略折算分数导致相关 check 部分失败时评分不准确的问题。

### 发布

- 更新版本号到 `0.26.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub/Railway 镜像渠道不随常规发布自动执行；如需要，单独走“镜像渠道发布”流程。
