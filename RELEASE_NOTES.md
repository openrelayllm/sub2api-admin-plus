# Release Notes

## v0.16.0 - 2026-06-26

### 新增

- 集成 Mihomo core，本地 `./scripts/start-dev.sh` 会按需下载并复用 `.local/bin/mihomo`。
- DockerHub/Railway 镜像内置 `/app/bin/mihomo`，生产环境无需额外安装本地 Docker。
- 代理节点检测结果写入 Redis 缓存，刷新页面后继续显示最近的出口 IP、延迟和错误信息。
- 代理节点支持单节点检测和一键检测，前端逐行展示检测进度与结果。

### 改进

- 代理节点检测统一通过独立 Mihomo runtime 实测出口，避免受本机 Clash/Mihomo fake-ip DNS 影响。
- 仅对节点配置中直接写死的保留 IP 做快速失败，其余域名和真实连接交给 Mihomo core 验证。
- 代理运行时清理、节点切换预算、频率限制和审计过滤能力更完整。

### 修复

- 修复 fake-ip 场景下后端系统 DNS 预解析导致的节点误判。
- 修复页面刷新后节点检测状态丢失的问题。
- 修复 New API 注册状态探测超时或邮箱验证码提示时的直接注册流程。

### 发布

- 更新版本号到 `0.16.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub 镜像继续由 GitHub Actions 发布，不依赖本地 Docker。
