# Release Notes

## v0.13.0 - 2026-06-25

### 新增

- 新增采集候选“一键识别全部”能力，支持通过流式进度批量判断 new-api / sub2api 类型。
- 新增采集候选“批量加入目录”能力，可将已识别候选批量沉淀到站点目录。
- 新增站点目录批量加入流式接口与前端进度反馈，便于处理大量候选站点。

### 改进

- 采集分类改为并发流式写入，降低慢站点探测对整体采集进度的阻塞。
- 优化公开接口探测超时控制，避免单个上游接口拖慢批量识别。
- 供应商会话 profile 探测支持从 sub2api 失败路径回退到 new-api，并保留 fallback 诊断信息。
- 余额刷新错误补充 endpoint、状态码、响应类型和响应摘录，便于定位上游接口不匹配或登录态问题。
- 采集列表补充类型筛选快捷入口，支持按 new-api、sub2api、未知类型快速筛选。

### 修复

- 修复已应用 `179_admin_plus_site_catalog.sql` 的历史校验兼容，避免旧环境升级时 checksum 阻断。
- 修复 NewAPI 会话错误缺少 HTTP 响应诊断的问题。
- 修复余额刷新错误文案丢失结构化 metadata 的问题。

### 发布

- 更新版本号到 `0.13.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub 镜像继续由 GitHub Actions 发布，不依赖本地 Docker。
