# Release Notes

## v0.11.0 - 2026-06-23

### 新增

- 新增 Admin Plus 调度中心 `/admin/scheduler`，旧 `/admin/collection/scheduler` 和 `/admin/operations/scheduler` 兼容重定向到新入口。
- 新增调度中心计划、运行、步骤、attempt、智能动作和设置表，支持 run/step 持久化、租约 claim、取消、失败重试和运行详情审计。
- 新增调度中心 API，覆盖中心状态、计划配置、运行记录、步骤操作、供应商自动化 Checklist、智能动作和全局设置。
- 新增调度中心前端工作台、计划面板、运行详情、供应商自动化矩阵、Checklist 弹窗和状态展示组件。

### 改进

- 手动调度默认提交异步 run，dry-run 继续保留同步预检能力，避免请求线程直接执行长耗时采集。
- 渠道最佳选择会基于当前候选分组投影最新检测结果，未绑定本地账号的候选明确标记为不可调度。
- 供应商编辑的充值倍率输入允许任意小数步进，避免浏览器 number step 限制阻断精细倍率。
- 调度中心路线和侧边栏文案收敛到当前运营入口，减少旧采集页面暴露。

### 发布

- 更新版本号到 `0.11.0`。
- GitHub Release 继续只发布 Linux 产物：`linux_amd64`、`linux_arm64` 和 `checksums.txt`。
- DockerHub 镜像继续由 GitHub Actions 发布，不依赖本地 Docker。
