# Release Notes

## v0.9.9 - 2026-06-23

### 新增

- New API provider adapter 支持读取充值记录，接入 `/api/user/topup/self` 并同步到供应商成本台账。
- New API provider adapter 支持读取用量成本记录，接入 `/api/log/self` 并同步真实消耗成本。
- New API quota 统一按 `500000 quota units = 1 USD` 换算，Key 创建、用户余额、充值记录和历史 QTA 余额读取保持同一 USD 口径。
- 供应商支持配置充值倍率 `recharge_multiplier`，充值入账、成本台账和成本快照统一记录折算后的实际付款金额。
- 供应商最佳渠道按 OpenAI / Claude / Gemini 协议分别返回推荐结果，前端支持按协议筛选、展示多协议最佳渠道和可用协议提示。
- 供应商列表支持快速切换运行状态和健康状态，并把余额、成本快照、会话状态、凭据状态合并到更紧凑的运营视图。

### 修复

- 供应商、余额快照、余额事件和成本台账读取时会把历史 `QTA` / `CNY` 口径归一为 `USD`，避免前端显示混币种成本。
- 成本总览和成本台账只汇总 USD 数据，`SUCCESS` 状态充值会计入完成充值。
- 修复本地 Sub2API 账号不存在时的识别逻辑，兼容 404、no rows、not exist 和中文不存在类错误。
- New API 充值、token、profile 原始响应快照继续过滤 key、token、secret、password、cookie 等敏感字段。
- 移除独立公告路由入口，旧公告路径统一回到供应商运营页，减少不可用运营入口干扰。
- 新增迁移 `173_admin_plus_supplier_recharge_multiplier`，为供应商、充值交易、成本台账和成本快照补齐充值倍率与实际付款字段。

### 发布

- 更新版本号到 `0.9.9`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub 镜像继续由 GitHub Actions 发布，不依赖本地 Docker。
