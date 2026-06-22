# Release Notes

## v0.9.8 - 2026-06-22

### 新增

- 供应商新增 channel check 能力：按 active group 选择候选通道，读取远端 channel monitor，执行 OpenAI Responses 探测并保存检测快照。
- 新增 channel check API：最佳通道列表、供应商历史检测、单 group 复测、异步批量检测，以及本地 Sub2API account 调度启停。
- 供应商页新增最佳通道列、批量检测入口、group 级通道检测结果、快速补齐绑定并加入调度的操作流。
- 成本对账新增总账概览，按币种聚合最新供应商快照，展示充值、权益、用量、退款、调整、预期余额和实际余额差异。
- Scheduler 支持可选自动 channel check 任务，可通过 `ADMIN_PLUS_CHANNEL_CHECKS_SCHEDULER_ENABLED` 开启。

### 修复

- Chrome 扩展不再从页面文本解析余额，避免与后端会话余额同步口径重复。
- Sub2API 扩展采集的默认 `api_base_url` 改为站点 origin，避免自动追加 `/api` 导致供应商 API 根地址错误。
- 扩展弹窗对后台余额同步失败展示脱敏诊断，隐藏 authorization、cookie、token、secret、password 等敏感片段。
- 供应商会话 profile 兼容远端 API base URL，减少直登会话和本地账号绑定时的地址不一致。

### 发布

- 更新版本号到 `0.9.8`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub 镜像继续由 GitHub Actions 发布，不依赖本地 Docker。
