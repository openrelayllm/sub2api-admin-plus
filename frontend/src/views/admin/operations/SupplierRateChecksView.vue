<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div class="flex flex-wrap items-center gap-2">
            <div class="inline-flex overflow-hidden rounded-md border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-900">
              <button
                type="button"
                class="rounded px-3 py-1.5 text-sm font-medium"
                :class="protocol === 'openai' ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-200' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                @click="protocol = 'openai'"
              >
                OpenAI
              </button>
              <button
                type="button"
                class="rounded px-3 py-1.5 text-sm font-medium"
                :class="protocol === 'anthropic' ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-200' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                @click="protocol = 'anthropic'"
              >
                Anthropic
              </button>
            </div>
            <span class="badge badge-gray">渠道 {{ stats.total }}</span>
            <span class="badge badge-success">通畅 {{ stats.available }}</span>
            <span class="badge badge-primary">调度 {{ stats.scheduled }}</span>
            <span class="badge" :class="stats.changed > 0 ? 'badge-warning' : 'badge-gray'">变动 {{ stats.changed }}</span>
          </div>

          <div class="flex flex-wrap items-end gap-2">
            <label class="block w-28">
              <span class="input-label">范围</span>
              <select v-model="mode" class="input h-9">
                <option value="best">最佳</option>
                <option value="all">全部</option>
              </select>
            </label>
            <label class="block min-w-[220px]">
              <span class="input-label">本地分组</span>
              <select v-model.number="selectedLocalGroupID" class="input h-9">
                <option value="">选择本地分组</option>
                <option
                  v-for="group in localGroups"
                  :key="group.id"
                  :value="group.id"
                >
                  {{ group.name }} #{{ group.id }}
                </option>
              </select>
            </label>
            <button type="button" class="btn btn-secondary h-9 px-2 md:px-3" :disabled="schedulerSubmitting" title="提交分组同步调度" @click="syncGroups">
              <Icon name="sync" size="sm" :class="{ 'animate-spin': schedulerSubmitting }" />
              <span class="hidden md:inline">同步倍率</span>
            </button>
            <button type="button" class="btn btn-secondary h-9 px-2 md:px-3" :disabled="schedulerSubmitting" title="提交渠道通畅检测调度" @click="checkChannels">
              <Icon name="beaker" size="sm" :class="{ 'animate-spin': schedulerSubmitting }" />
              <span class="hidden md:inline">检测通畅</span>
            </button>
            <button type="button" class="btn btn-secondary h-9 px-2 md:px-3" :disabled="loading" title="刷新倍率检测列表" @click="loadRows">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
              <span class="hidden md:inline">刷新</span>
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <div v-if="error" class="mb-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-200">
          {{ error }}
        </div>

        <div class="overflow-x-auto rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <table class="min-w-full table-fixed text-left text-sm">
            <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-900 dark:text-dark-400">
              <tr class="border-b border-gray-100 dark:border-dark-700">
                <th class="w-[280px] px-4 py-3 font-medium">渠道</th>
                <th class="w-[120px] px-4 py-3 text-right font-medium">倍率</th>
                <th class="w-[128px] px-4 py-3 font-medium">变动</th>
                <th class="w-[180px] px-4 py-3 font-medium">通畅</th>
                <th class="w-[220px] px-4 py-3 font-medium">本地分组</th>
                <th class="w-[100px] px-4 py-3 font-medium">调度</th>
                <th class="w-[132px] px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading && rows.length === 0">
                <td colspan="7" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">加载中...</td>
              </tr>
              <tr v-else-if="rows.length === 0">
                <td colspan="7" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">暂无 {{ protocolLabel(protocol) }} 渠道</td>
              </tr>
              <template v-else>
                <tr
                  v-for="row in rows"
                  :key="rowKey(row)"
                  class="border-b border-gray-50 last:border-0 dark:border-dark-700/70"
                >
                  <td class="px-4 py-3">
                    <div class="min-w-0">
                      <div class="flex min-w-0 items-center gap-2">
                        <span class="badge shrink-0" :class="row.protocol === 'anthropic' ? 'badge-purple' : 'badge-primary'">{{ protocolLabel(row.protocol) }}</span>
                        <span class="truncate font-medium text-gray-900 dark:text-gray-100" :title="row.supplier_name">{{ row.supplier_name }}</span>
                      </div>
                      <div class="mt-1 flex min-w-0 items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                        <span class="truncate" :title="row.group_name">{{ row.group_name || '-' }}</span>
                        <span class="font-mono">#{{ row.supplier_group_id }}</span>
                      </div>
                    </div>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <span :class="rowRateClass(row)">{{ formatMultiplier(rowCostMultiplier(row)) }}</span>
                    <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ formatMultiplier(row.effective_rate_multiplier) }}</div>
                  </td>
                  <td class="px-4 py-3">
                    <div class="text-sm font-medium" :class="changeClass(row)">{{ changeLabel(row) }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ row.changed_at ? formatDateTime(row.changed_at) : '-' }}</div>
                  </td>
                  <td class="px-4 py-3">
                    <span class="badge" :class="probeStatusClass(row.probe_status)" :title="probeTitle(row)">{{ probeStatusLabel(row.probe_status) }}</span>
                    <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                      首 {{ formatLatency(row.first_token_ms) }} · 总 {{ formatLatency(row.duration_ms) }}
                    </div>
                  </td>
                  <td class="px-4 py-3">
                    <div v-if="rowLocalGroupLabels(row).length > 0" class="flex max-w-[210px] flex-wrap gap-1">
                      <span
                        v-for="label in rowLocalGroupLabels(row)"
                        :key="`${rowKey(row)}:${label}`"
                        class="inline-flex max-w-[96px] rounded-md border border-primary-200 bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-200"
                        :title="label"
                      >
                        <span class="truncate">{{ label }}</span>
                      </span>
                    </div>
                    <span v-else class="badge badge-gray">未绑定</span>
                  </td>
                  <td class="px-4 py-3">
                    <span class="badge" :class="row.local_account_schedulable ? 'badge-success' : 'badge-gray'">
                      {{ row.local_account_schedulable ? '调度中' : '未调度' }}
                    </span>
                  </td>
                  <td class="px-4 py-3">
                    <div class="flex justify-end gap-1">
                      <button
                        type="button"
                        class="btn btn-secondary btn-sm h-8 w-8 px-0"
                        :disabled="Boolean(actionKey)"
                        title="复测该渠道"
                        @click="probeRow(row)"
                      >
                        <Icon name="beaker" size="xs" :class="{ 'animate-spin': isActionRunning(row, 'probe') }" />
                        <span class="sr-only">复测</span>
                      </button>
                      <button
                        type="button"
                        class="btn btn-secondary btn-sm h-8 w-8 px-0"
                        :disabled="Boolean(actionKey) || !selectedGroupID()"
                        title="加入选择的本地分组"
                        @click="bindRowToLocalGroup(row)"
                      >
                        <Icon name="link" size="xs" :class="{ 'animate-spin': isActionRunning(row, 'bind') }" />
                        <span class="sr-only">加入分组</span>
                      </button>
                      <button
                        type="button"
                        class="btn btn-secondary btn-sm h-8 w-8 px-0"
                        :disabled="Boolean(actionKey)"
                        :title="row.local_account_schedulable ? '暂停调度' : '加入调度'"
                        @click="row.local_account_schedulable ? pauseRow(row) : scheduleRow(row)"
                      >
                        <Icon :name="row.local_account_schedulable ? 'ban' : 'play'" size="xs" :class="{ 'animate-spin': isActionRunning(row, row.local_account_schedulable ? 'pause' : 'schedule') }" />
                        <span class="sr-only">{{ row.local_account_schedulable ? '暂停调度' : '加入调度' }}</span>
                      </button>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { createSchedulerRun, enableSupplierChannelScheduling, listSupplierChannelCheckOverview, listSuppliers, pauseSupplierChannelScheduling, probeSupplierChannel } from '@/api/admin/adminPlus'
import type { ExtensionTaskType, Supplier, SupplierChannelCheckOverviewMode, SupplierChannelCheckOverviewRow, SupplierChannelCheckProtocol, SupplierChannelCheckSnapshot, SupplierChannelProbeStatus } from '@/api/admin/adminPlus'
import { groupsAPI } from '@/api/admin/groups'
import type { AdminGroup, GroupPlatform } from '@/types'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from './SupplierAccountsUtils'

type Protocol = Extract<SupplierChannelCheckProtocol, 'openai' | 'anthropic'>

const appStore = useAppStore()
const protocol = ref<Protocol>('openai')
const mode = ref<SupplierChannelCheckOverviewMode>('best')
const selectedLocalGroupID = ref<number | ''>('')
const rows = ref<SupplierChannelCheckOverviewRow[]>([])
const localGroups = ref<AdminGroup[]>([])
const suppliersByID = ref<Record<number, Supplier | undefined>>({})
const loading = ref(false)
const schedulerSubmitting = ref(false)
const actionKey = ref('')
const error = ref('')

const stats = computed(() => ({
  total: rows.value.length,
  available: rows.value.filter((row) => row.probe_status === 'available').length,
  scheduled: rows.value.filter((row) => row.local_account_schedulable).length,
  changed: rows.value.filter((row) => Boolean(row.change_event_id)).length
}))

async function loadRows() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const [overview, suppliers] = await Promise.all([
      listSupplierChannelCheckOverview({
        protocol: protocol.value,
        mode: mode.value
      }),
      listSuppliers({ limit: 1000 })
    ])
    rows.value = overview.items || []
    suppliersByID.value = Object.fromEntries((suppliers.items || []).map((supplier) => [supplier.id, supplier]))
  } catch (err) {
    rows.value = []
    error.value = (err as { message?: string }).message || '加载倍率检测失败'
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}

async function loadLocalGroups() {
  try {
    localGroups.value = await groupsAPI.getAll(groupPlatform())
  } catch (err) {
    localGroups.value = []
    appStore.showError((err as { message?: string }).message || '加载本地分组失败')
  }
}

async function runSchedulerTask(taskType: ExtensionTaskType) {
  if (schedulerSubmitting.value) return
  schedulerSubmitting.value = true
  error.value = ''
  try {
    const run = await createSchedulerRun({
      mode: 'manual',
      task_types: [taskType],
      window_minutes: 10
    })
    appStore.showSuccess(`调度任务已提交 #${run.id}`)
    window.setTimeout(() => {
      void loadRows()
    }, 1500)
  } catch (err) {
    error.value = (err as { message?: string }).message || '提交调度任务失败'
    appStore.showError(error.value)
  } finally {
    schedulerSubmitting.value = false
  }
}

async function syncGroups() {
  await runSchedulerTask('fetch_groups')
}

async function checkChannels() {
  await runSchedulerTask('check_supplier_channels')
}

async function bindRowToLocalGroup(row: SupplierChannelCheckOverviewRow) {
  const localGroupID = selectedGroupID()
  if (!localGroupID) {
    appStore.showWarning('请选择本地分组')
    return
  }
  await mutateRow(`bind:${rowKey(row)}`, async () => {
    return pauseSupplierChannelScheduling(row.supplier_id, row.supplier_group_id, {
      local_account_group_ids: [localGroupID]
    })
  }, '已加入本地分组', '加入本地分组失败')
}

async function scheduleRow(row: SupplierChannelCheckOverviewRow) {
  const localGroupID = selectedGroupID()
  const groupIDs = localGroupID ? [localGroupID] : []
  if (groupIDs.length === 0 && (row.local_account_group_ids || []).length === 0) {
    appStore.showWarning('请选择本地分组后再加入调度')
    return
  }
  await mutateRow(`schedule:${rowKey(row)}`, async () => {
    return enableSupplierChannelScheduling(row.supplier_id, row.supplier_group_id, {
      local_account_group_ids: groupIDs.length > 0 ? groupIDs : undefined
    })
  }, '已加入调度', '加入调度失败')
}

async function pauseRow(row: SupplierChannelCheckOverviewRow) {
  await mutateRow(`pause:${rowKey(row)}`, async () => {
    return pauseSupplierChannelScheduling(row.supplier_id, row.supplier_group_id)
  }, '已暂停调度', '暂停调度失败')
}

async function probeRow(row: SupplierChannelCheckOverviewRow) {
  if (actionKey.value) return
  const key = `probe:${rowKey(row)}`
  actionKey.value = key
  error.value = ''
  try {
    await probeSupplierChannel(row.supplier_id, {
      supplier_group_id: row.supplier_group_id,
      auto_pause_on_failure: false
    })
    await loadRows()
    appStore.showSuccess('渠道检测完成')
  } catch (err) {
    error.value = (err as { message?: string }).message || '渠道检测失败'
    appStore.showError(error.value)
  } finally {
    if (actionKey.value === key) {
      actionKey.value = ''
    }
  }
}

async function mutateRow(
  key: string,
  action: () => Promise<SupplierChannelCheckSnapshot>,
  successMessage: string,
  failureMessage: string
) {
  if (actionKey.value) return
  actionKey.value = key
  error.value = ''
  try {
    await action()
    await loadRows()
    appStore.showSuccess(successMessage)
  } catch (err) {
    error.value = (err as { message?: string }).message || failureMessage
    appStore.showError(error.value)
  } finally {
    if (actionKey.value === key) {
      actionKey.value = ''
    }
  }
}

function groupPlatform(): GroupPlatform {
  return protocol.value === 'anthropic' ? 'anthropic' : 'openai'
}

function selectedGroupID(): number {
  const id = Number(selectedLocalGroupID.value || 0)
  return Number.isFinite(id) && id > 0 ? id : 0
}

function rowKey(row: SupplierChannelCheckOverviewRow): string {
  return `${row.supplier_id}:${row.supplier_group_id}`
}

function isActionRunning(row: SupplierChannelCheckOverviewRow, action: string): boolean {
  return actionKey.value === `${action}:${rowKey(row)}`
}

function protocolLabel(value?: SupplierChannelCheckProtocol | string): string {
  if (value === 'anthropic') return 'Anthropic'
  return 'OpenAI'
}

function rowCostMultiplier(row: SupplierChannelCheckOverviewRow): number {
  return actualCostMultiplier(row.effective_rate_multiplier, supplierRechargeMultiplier(row.supplier_id))
}

function rowRateClass(row: SupplierChannelCheckOverviewRow): string {
  return rateMultiplierTextClass(rowCostMultiplier(row), row.protocol)
}

function rowLocalGroupLabels(row: SupplierChannelCheckOverviewRow): string[] {
  const ids = Array.isArray(row.local_account_group_ids) ? row.local_account_group_ids : []
  const names = Array.isArray(row.local_account_group_names) ? row.local_account_group_names : []
  const length = Math.max(ids.length, names.length)
  const labels: string[] = []
  for (let index = 0; index < length; index++) {
    labels.push(names[index] || (ids[index] ? `分组 #${ids[index]}` : ''))
  }
  return labels.filter(Boolean)
}

function changeLabel(row: SupplierChannelCheckOverviewRow): string {
  if (!row.change_event_id) return '-'
  if (row.change_direction === 'new') return '新增'
  const direction = row.change_direction === 'increase' ? '升高' : '降低'
  const percent = typeof row.change_percent === 'number' ? ` ${Math.abs(row.change_percent).toFixed(1)}%` : ''
  return `${direction}${percent}`
}

function changeClass(row: SupplierChannelCheckOverviewRow): string {
  if (row.change_direction === 'decrease') return 'text-emerald-700 dark:text-emerald-300'
  if (row.change_direction === 'increase') return 'text-rose-700 dark:text-rose-300'
  if (row.change_direction === 'new') return 'text-primary-700 dark:text-primary-300'
  return 'text-gray-500 dark:text-dark-400'
}

function probeTitle(row: SupplierChannelCheckOverviewRow): string {
  return [
    probeStatusLabel(row.probe_status),
    row.error_message,
    row.captured_at ? `检测时间 ${formatDateTime(row.captured_at)}` : ''
  ].filter(Boolean).join(' · ')
}

function supplierRechargeMultiplier(supplierID?: number | null): number {
  if (!supplierID) return 1
  return normalizedRechargeMultiplier(suppliersByID.value[supplierID]?.recharge_multiplier)
}

function normalizedRechargeMultiplier(value?: number | null): number {
  const multiplier = Number(value || 0)
  return Number.isFinite(multiplier) && multiplier > 0 ? multiplier : 1
}

function actualCostMultiplier(rate?: number | null, rechargeMultiplier?: number | null): number {
  const usageRate = Number(rate || 0)
  if (!Number.isFinite(usageRate) || usageRate <= 0) return 0
  return usageRate / normalizedRechargeMultiplier(rechargeMultiplier)
}

function formatMultiplier(value?: number | null): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  return `${value.toFixed(4).replace(/\.?0+$/, '')}x`
}

function formatLatency(value?: number | null): string {
  if (typeof value !== 'number' || value <= 0) return '-'
  if (value >= 1000) return `${(value / 1000).toFixed(value >= 10000 ? 0 : 1)}s`
  return `${Math.round(value)}ms`
}

function rateMultiplierTextClass(value?: number | null, protocolValue?: SupplierChannelCheckProtocol): string {
  const base = 'inline-flex items-center justify-end rounded-md px-1.5 py-0.5 text-lg font-extrabold leading-tight ring-1 whitespace-nowrap'
  if (protocolValue === 'openai' && typeof value === 'number' && Number.isFinite(value) && value > 0.1) {
    return `${base} bg-rose-50 text-rose-700 ring-rose-200 dark:bg-rose-950/50 dark:text-rose-300 dark:ring-rose-800/60`
  }
  return `${base} bg-green-50 text-green-800 ring-green-200 dark:bg-green-950/50 dark:text-green-300 dark:ring-green-800/60`
}

function probeStatusLabel(status?: SupplierChannelProbeStatus): string {
  if (status === 'available') return '可用'
  if (status === 'slow_first_token') return '首 token 慢'
  if (status === 'slow_total') return '总耗时慢'
  if (status === 'request_error') return '请求失败'
  if (status === 'remote_unavailable') return '远端异常'
  if (status === 'no_local_account') return '未绑定账号'
  if (status === 'probe_failed') return '检测失败'
  return '未检测'
}

function probeStatusClass(status?: SupplierChannelProbeStatus): string {
  if (status === 'available') return 'badge-success'
  if (status === 'slow_first_token' || status === 'slow_total' || status === 'remote_unavailable') return 'badge-warning'
  if (status === 'request_error' || status === 'probe_failed') return 'badge-danger'
  return 'badge-gray'
}

watch(protocol, () => {
  selectedLocalGroupID.value = ''
  void loadLocalGroups()
  void loadRows()
})

watch(mode, () => {
  void loadRows()
})

onMounted(() => {
  void loadLocalGroups()
  void loadRows()
})
</script>
