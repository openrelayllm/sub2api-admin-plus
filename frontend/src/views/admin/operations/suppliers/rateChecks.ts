import { computed, onMounted, watch } from 'vue'
import { createSchedulerRun, enableSupplierChannelScheduling, listSupplierChannelCheckOverview, pauseSupplierChannelScheduling, probeSupplierChannel } from '@/api/admin/adminPlus'
import { groupsAPI } from '@/api/admin/groups'
import type { GroupPlatform } from '@/types'
import type { ExtensionTaskType, SupplierChannelCheckOverviewRow, SupplierChannelCheckSnapshot } from '@/api/admin/adminPlus'
import type { RateCheckProtocol } from './types'
import { ctxFn, ctxValue } from './ctxProxy'

export function attachSupplierRateChecks(ctx: any) {
  const appStore = ctxValue(ctx, 'appStore')
  const rateCheckRows = ctxValue(ctx, 'rateCheckRows')
  const rateCheckLocalGroups = ctxValue(ctx, 'rateCheckLocalGroups')
  const rateCheckProtocol = ctxValue(ctx, 'rateCheckProtocol')
  const rateCheckMode = ctxValue(ctx, 'rateCheckMode')
  const rateCheckSelectedLocalGroupID = ctxValue(ctx, 'rateCheckSelectedLocalGroupID')
  const rateCheckLoading = ctxValue(ctx, 'rateCheckLoading')
  const rateCheckSchedulerSubmitting = ctxValue(ctx, 'rateCheckSchedulerSubmitting')
  const rateCheckActionKey = ctxValue(ctx, 'rateCheckActionKey')
  const rateCheckError = ctxValue(ctx, 'rateCheckError')
  const formatDateTime = ctxFn(ctx, 'formatDateTime')
  const actualCostMultiplier = ctxFn(ctx, 'actualCostMultiplier')
  const supplierRechargeMultiplier = ctxFn(ctx, 'supplierRechargeMultiplier')
  const channelProbeStatusLabel = ctxFn(ctx, 'channelProbeStatusLabel')
  const rateMultiplierTextClass = ctxFn(ctx, 'rateMultiplierTextClass')
  const mergeChannelCheckSnapshots = ctxFn(ctx, 'mergeChannelCheckSnapshots')
  const upsertSupplierBestChannelSnapshot = ctxFn(ctx, 'upsertSupplierBestChannelSnapshot')
  const loadBestChannelChecks = ctxFn(ctx, 'loadBestChannelChecks')

  const rateCheckStats = computed(() => {
    const rows = rateCheckRows.value as SupplierChannelCheckOverviewRow[]
    return {
      total: rows.length,
      available: rows.filter((row) => row.probe_status === 'available').length,
      scheduled: rows.filter((row) => row.local_account_schedulable).length,
      changed: rows.filter((row) => Boolean(row.change_event_id)).length
    }
  })

  async function loadSupplierRateChecks() {
    if (rateCheckLoading.value) return
    rateCheckLoading.value = true
    rateCheckError.value = ''
    try {
      const result = await listSupplierChannelCheckOverview({
        protocol: rateCheckProtocol.value,
        mode: rateCheckMode.value
      })
      rateCheckRows.value = result.items || []
    } catch (error) {
      rateCheckRows.value = []
      rateCheckError.value = (error as { message?: string }).message || '加载倍率检测失败'
      appStore.showError(rateCheckError.value)
    } finally {
      rateCheckLoading.value = false
    }
  }

  async function loadRateCheckLocalGroups() {
    try {
      rateCheckLocalGroups.value = await groupsAPI.getAll(rateCheckGroupPlatform())
    } catch (error) {
      rateCheckLocalGroups.value = []
      appStore.showError((error as { message?: string }).message || '加载本地分组失败')
    }
  }

  async function runRateCheckSchedulerTask(taskType: ExtensionTaskType) {
    if (rateCheckSchedulerSubmitting.value) return
    rateCheckSchedulerSubmitting.value = true
    rateCheckError.value = ''
    try {
      const run = await createSchedulerRun({
        mode: 'manual',
        task_types: [taskType],
        window_minutes: 10
      })
      appStore.showSuccess(`调度任务已提交 #${run.id}`)
      window.setTimeout(() => {
        void loadSupplierRateChecks()
      }, 1500)
    } catch (error) {
      rateCheckError.value = (error as { message?: string }).message || '提交调度任务失败'
      appStore.showError(rateCheckError.value)
    } finally {
      rateCheckSchedulerSubmitting.value = false
    }
  }

  async function syncRateCheckGroups() {
    await runRateCheckSchedulerTask('fetch_groups')
  }

  async function checkRateCheckChannels() {
    await runRateCheckSchedulerTask('check_supplier_channels')
  }

  async function bindRateCheckRowToLocalGroup(row: SupplierChannelCheckOverviewRow) {
    const localGroupID = selectedRateCheckLocalGroupID()
    if (!localGroupID) {
      appStore.showWarning('请选择本地分组')
      return
    }
    await mutateRateCheckRow(row, `bind:${rateCheckRowKey(row)}`, async () => {
      return pauseSupplierChannelScheduling(row.supplier_id, row.supplier_group_id, {
        local_account_group_ids: [localGroupID]
      })
    }, '已加入本地分组', '加入本地分组失败')
  }

  async function scheduleRateCheckRow(row: SupplierChannelCheckOverviewRow) {
    const localGroupID = selectedRateCheckLocalGroupID()
    const groupIDs = localGroupID ? [localGroupID] : []
    if (groupIDs.length === 0 && (row.local_account_group_ids || []).length === 0) {
      appStore.showWarning('请选择本地分组后再加入调度')
      return
    }
    await mutateRateCheckRow(row, `schedule:${rateCheckRowKey(row)}`, async () => {
      return enableSupplierChannelScheduling(row.supplier_id, row.supplier_group_id, {
        local_account_group_ids: groupIDs.length > 0 ? groupIDs : undefined
      })
    }, '已加入调度', '加入调度失败')
  }

  async function pauseRateCheckRow(row: SupplierChannelCheckOverviewRow) {
    await mutateRateCheckRow(row, `pause:${rateCheckRowKey(row)}`, async () => {
      return pauseSupplierChannelScheduling(row.supplier_id, row.supplier_group_id)
    }, '已暂停调度', '暂停调度失败')
  }

  async function probeRateCheckRow(row: SupplierChannelCheckOverviewRow) {
    if (rateCheckActionKey.value) return
    const actionKey = `probe:${rateCheckRowKey(row)}`
    rateCheckActionKey.value = actionKey
    rateCheckError.value = ''
    try {
      const result = await probeSupplierChannel(row.supplier_id, {
        supplier_group_id: row.supplier_group_id,
        auto_pause_on_failure: false
      })
      mergeChannelCheckSnapshots(result.items)
      const current = result.items.find((item) => item.supplier_group_id === row.supplier_group_id)
      if (current) {
        upsertSupplierBestChannelSnapshot(current)
      }
      await Promise.all([loadSupplierRateChecks(), loadBestChannelChecks([row.supplier_id])])
      appStore.showSuccess('渠道检测完成')
    } catch (error) {
      rateCheckError.value = (error as { message?: string }).message || '渠道检测失败'
      appStore.showError(rateCheckError.value)
    } finally {
      if (rateCheckActionKey.value === actionKey) {
        rateCheckActionKey.value = ''
      }
    }
  }

  async function mutateRateCheckRow(
    row: SupplierChannelCheckOverviewRow,
    actionKey: string,
    action: () => Promise<SupplierChannelCheckSnapshot>,
    successMessage: string,
    failureMessage: string
  ) {
    if (rateCheckActionKey.value) return
    rateCheckActionKey.value = actionKey
    rateCheckError.value = ''
    try {
      const snapshot = await action()
      mergeChannelCheckSnapshots([snapshot])
      upsertSupplierBestChannelSnapshot(snapshot)
      await Promise.all([loadSupplierRateChecks(), loadBestChannelChecks([row.supplier_id])])
      appStore.showSuccess(successMessage)
    } catch (error) {
      rateCheckError.value = (error as { message?: string }).message || failureMessage
      appStore.showError(rateCheckError.value)
    } finally {
      if (rateCheckActionKey.value === actionKey) {
        rateCheckActionKey.value = ''
      }
    }
  }

  function rateCheckGroupPlatform(): GroupPlatform {
    return rateCheckProtocol.value === 'anthropic' ? 'anthropic' : 'openai'
  }

  function selectedRateCheckLocalGroupID(): number {
    const id = Number(rateCheckSelectedLocalGroupID.value || 0)
    return Number.isFinite(id) && id > 0 ? id : 0
  }

  function rateCheckRowKey(row: SupplierChannelCheckOverviewRow): string {
    return `${row.supplier_id}:${row.supplier_group_id}`
  }

  function isRateCheckRowActionRunning(row: SupplierChannelCheckOverviewRow, action: string): boolean {
    return rateCheckActionKey.value === `${action}:${rateCheckRowKey(row)}`
  }

  function rateCheckProtocolLabel(protocol?: RateCheckProtocol | string): string {
    if (protocol === 'anthropic') return 'Anthropic'
    return 'OpenAI'
  }

  function rateCheckRowCostMultiplier(row: SupplierChannelCheckOverviewRow): number {
    return actualCostMultiplier(row.effective_rate_multiplier, supplierRechargeMultiplier(row.supplier_id))
  }

  function rateCheckRowRateClass(row: SupplierChannelCheckOverviewRow): string {
    return rateMultiplierTextClass(rateCheckRowCostMultiplier(row), row.protocol === 'anthropic' ? 'claude' : row.protocol, 'compact')
  }

  function rateCheckRowLocalGroupLabels(row: SupplierChannelCheckOverviewRow): string[] {
    const ids = Array.isArray(row.local_account_group_ids) ? row.local_account_group_ids : []
    const names = Array.isArray(row.local_account_group_names) ? row.local_account_group_names : []
    const length = Math.max(ids.length, names.length)
    const labels: string[] = []
    for (let index = 0; index < length; index++) {
      labels.push(names[index] || (ids[index] ? `分组 #${ids[index]}` : ''))
    }
    return labels.filter(Boolean)
  }

  function rateCheckChangeLabel(row: SupplierChannelCheckOverviewRow): string {
    if (!row.change_event_id) return '-'
    if (row.change_direction === 'new') return '新增'
    const direction = row.change_direction === 'increase' ? '升高' : '降低'
    const percent = typeof row.change_percent === 'number' ? ` ${Math.abs(row.change_percent).toFixed(1)}%` : ''
    return `${direction}${percent}`
  }

  function rateCheckChangeClass(row: SupplierChannelCheckOverviewRow): string {
    if (row.change_direction === 'decrease') return 'text-emerald-700 dark:text-emerald-300'
    if (row.change_direction === 'increase') return 'text-rose-700 dark:text-rose-300'
    if (row.change_direction === 'new') return 'text-primary-700 dark:text-primary-300'
    return 'text-gray-500 dark:text-dark-400'
  }

  function rateCheckProbeTitle(row: SupplierChannelCheckOverviewRow): string {
    return [
      channelProbeStatusLabel(row.probe_status),
      row.error_message,
      row.captured_at ? `检测时间 ${formatDateTime(row.captured_at)}` : ''
    ].filter(Boolean).join(' · ')
  }

  watch(rateCheckProtocol, () => {
    rateCheckSelectedLocalGroupID.value = ''
    void loadRateCheckLocalGroups()
    void loadSupplierRateChecks()
  })

  watch(rateCheckMode, () => {
    void loadSupplierRateChecks()
  })

  onMounted(() => {
    void loadRateCheckLocalGroups()
    void loadSupplierRateChecks()
  })

  Object.assign(ctx, {
    rateCheckStats,
    loadSupplierRateChecks,
    loadRateCheckLocalGroups,
    syncRateCheckGroups,
    checkRateCheckChannels,
    bindRateCheckRowToLocalGroup,
    scheduleRateCheckRow,
    pauseRateCheckRow,
    probeRateCheckRow,
    rateCheckGroupPlatform,
    selectedRateCheckLocalGroupID,
    rateCheckRowKey,
    isRateCheckRowActionRunning,
    rateCheckProtocolLabel,
    rateCheckRowCostMultiplier,
    rateCheckRowRateClass,
    rateCheckRowLocalGroupLabels,
    rateCheckChangeLabel,
    rateCheckChangeClass,
    rateCheckProbeTitle
  })
}
