# Release Notes

## v0.35.0 - 2026-07-03

### 新增

- 供应商通道检测新增 Overview 接口，聚合供应商分组、绑定本地账号、最新检测快照和倍率变动信息。
- 供应商列表新增倍率/通畅面板，支持 OpenAI 与 Anthropic 协议切换、最佳/全部范围切换、倍率变动查看、单渠道复测、加入本地分组、加入调度和暂停调度。
- NewAPI 供应商直登支持 access token 模式，可从 JSON、header block 或 `user_id:token` 形式解析 `access_token` 与 `New-Api-User`。
- 健康检测新增 Anthropic Messages 流式探测，默认模型为 `claude-sonnet-4-6`。

### 改进

- 通道检测根据协议选择 OpenAI Responses 或 Anthropic Messages 探测，并按协议使用默认探测模型。
- 通道调度支持传入本地账号分组 ID，绑定补全时会把选择的本地分组同步到本地账号。
- 浏览器扩展增强 NewAPI 会话识别，可从更多 localStorage 结构中提取用户 ID、用户名、邮箱、角色和状态。
- 供应商密钥 EnsureAll 支持向本地账号写入指定本地分组。

### 修复

- 修正 NewAPI access token 直登此前被强制要求用户名密码的问题。
- 修正 Anthropic/Claude 通道检测只能走 OpenAI 兼容探测模型的问题。

### 测试

- 增加 NewAPI access token 直登、Overview 最佳渠道筛选、指定本地分组调度和 Anthropic Messages 探测回归测试。

### 发布

- 更新版本号到 `0.35.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub/Railway 镜像渠道不随常规发布自动执行；如需要，单独走镜像渠道发布流程。
