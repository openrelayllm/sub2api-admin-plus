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
            <span class="badge badge-gray">账号 {{ stats.total }}</span>
            <span class="badge badge-primary">匹配 {{ stats.matched }}</span>
            <span class="badge badge-success">已改名 {{ stats.renamed }}</span>
            <span class="badge" :class="stats.pending > 0 ? 'badge-warning' : 'badge-gray'">待处理 {{ stats.pending }}</span>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <button type="button" class="btn btn-secondary h-9 px-2 md:px-3" :disabled="submitting" title="以当前本地账号 Key 反查供应商当前倍率" @click="syncRows">
              <Icon name="sync" size="sm" :class="{ 'animate-spin': submitting }" />
              <span class="hidden md:inline">同步账号倍率</span>
            </button>
            <button type="button" class="btn btn-secondary h-9 px-2 md:px-3" :disabled="submitting || stats.renameable === 0" title="把已匹配账号批量改为供应商名称加当前倍率" @click="renameAll">
              <Icon name="edit" size="sm" :class="{ 'animate-spin': submitting }" />
              <span class="hidden md:inline">更新全部名称</span>
            </button>
            <button type="button" class="btn btn-secondary h-9 px-2 md:px-3" :disabled="loading" title="刷新账号倍率同步列表" @click="loadRows">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
              <span class="hidden md:inline">刷新</span>
            </button>
            <button type="button" class="btn btn-danger h-9 px-2 md:px-3" :disabled="submitting" title="清空账号倍率同步历史" @click="clearHistory">
              <Icon name="trash" size="sm" />
              <span class="hidden md:inline">清空历史</span>
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
                <th class="w-[250px] px-4 py-3 font-medium">本地账号</th>
                <th class="w-[260px] px-4 py-3 font-medium">匹配供应商</th>
                <th class="w-[120px] px-4 py-3 text-right font-medium">当前倍率</th>
                <th class="w-[240px] px-4 py-3 font-medium">目标名称</th>
                <th class="w-[150px] px-4 py-3 font-medium">状态</th>
                <th class="w-[160px] px-4 py-3 font-medium">同步时间</th>
                <th class="w-[112px] px-4 py-3 text-right font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading && rows.length === 0">
                <td colspan="7" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">加载中...</td>
              </tr>
              <tr v-else-if="rows.length === 0">
                <td colspan="7" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">暂无 {{ protocolLabel(protocol) }} API Key 账号</td>
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
                        <span class="badge shrink-0" :class="row.local_account_platform === 'anthropic' ? 'badge-purple' : 'badge-primary'">{{ protocolLabel(row.local_account_platform) }}</span>
                        <span class="truncate font-medium text-gray-900 dark:text-gray-100" :title="row.local_account_name">{{ row.local_account_name }}</span>
                      </div>
                      <div class="mt-1 flex min-w-0 items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                        <span class="font-mono">#{{ row.local_sub2api_account_id }}</span>
                        <span v-if="row.key_last4" class="font-mono">****{{ row.key_last4 }}</span>
                      </div>
                    </div>
                  </td>
                  <td class="px-4 py-3">
                    <template v-if="row.supplier_id">
                      <div class="min-w-0">
                        <div class="truncate font-medium text-gray-900 dark:text-gray-100" :title="row.supplier_name">{{ row.supplier_name }}</div>
                        <div class="mt-1 flex min-w-0 items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                          <span class="truncate" :title="row.supplier_group_name">{{ row.supplier_group_name || '-' }}</span>
                          <span class="font-mono">#{{ row.supplier_group_id }}</span>
                        </div>
                      </div>
                    </template>
                    <span v-else class="badge badge-gray">未匹配</span>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <span v-if="row.effective_rate_multiplier > 0" class="font-semibold" :class="rateClass(row)">{{ rateText(row.effective_rate_multiplier) }}</span>
                    <span v-else class="text-gray-400 dark:text-dark-500">-</span>
                  </td>
                  <td class="px-4 py-3">
                    <div class="truncate text-sm text-gray-900 dark:text-gray-100" :title="row.target_account_name || ''">{{ row.target_account_name || '-' }}</div>
                    <div v-if="row.renamed && row.history?.new_account_name" class="mt-1 truncate text-xs text-emerald-600 dark:text-emerald-300" :title="row.history.new_account_name">已更新为 {{ row.history.new_account_name }}</div>
                  </td>
                  <td class="px-4 py-3">
                    <span class="badge" :class="statusClass(row.status)" :title="errorTitle(row)">{{ statusLabel(row.status) }}</span>
                    <div v-if="row.error_message" class="mt-1 max-w-[140px] truncate text-xs text-gray-500 dark:text-dark-400" :title="row.error_message">{{ row.error_message }}</div>
                  </td>
                  <td class="px-4 py-3 text-xs text-gray-500 dark:text-dark-400">
                    {{ formatDateTime(row.synced_at) }}
                  </td>
                  <td class="px-4 py-3">
                    <div class="flex justify-end gap-1">
                      <button
                        type="button"
                        class="btn btn-secondary btn-sm h-8 w-8 px-0"
                        :disabled="Boolean(actionKey)"
                        title="单个账号重试"
                        @click="retryRow(row)"
                      >
                        <Icon name="refresh" size="xs" :class="{ 'animate-spin': isActionRunning(row, 'retry') }" />
                        <span class="sr-only">重试</span>
                      </button>
                      <button
                        type="button"
                        class="btn btn-secondary btn-sm h-8 w-8 px-0"
                        :disabled="Boolean(actionKey) || !canRename(row)"
                        title="更新该账号名称"
                        @click="renameRow(row)"
                      >
                        <Icon name="edit" size="xs" :class="{ 'animate-spin': isActionRunning(row, 'rename') }" />
                        <span class="sr-only">更新名称</span>
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
import { clearAccountRateSyncHistory, listAccountRateSyncRows, renameAccountRateSyncRow, renameMatchedAccountRateSyncRows, retryAccountRateSyncRow, syncAccountRateRows } from '@/api/admin/adminPlus'
import type { AccountRateSyncRow, AccountRateSyncStatus, SupplierChannelCheckProtocol } from '@/api/admin/adminPlus'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from './SupplierAccountsUtils'

type Protocol = 'openai' | 'anthropic'

const appStore = useAppStore()
const protocol = ref<Protocol>('openai')
const rows = ref<AccountRateSyncRow[]>([])
const loading = ref(false)
const submitting = ref(false)
const actionKey = ref('')
const error = ref('')

const stats = computed(() => ({
  total: rows.value.length,
  matched: rows.value.filter((row) => row.status === 'matched' || row.status === 'renamed').length,
  renamed: rows.value.filter((row) => row.renamed || row.status === 'renamed').length,
  pending: rows.value.filter((row) => row.status === 'failed' || row.status === 'ambiguous' || row.status === 'not_found').length,
  renameable: rows.value.filter(canRename).length
}))

async function loadRows() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const result = await listAccountRateSyncRows({ protocol: protocol.value as SupplierChannelCheckProtocol, limit: 1000 })
    rows.value = result.items || []
  } catch (err) {
    rows.value = []
    error.value = (err as { message?: string }).message || '加载账号倍率同步失败'
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}

async function syncRows() {
  if (submitting.value) return
  submitting.value = true
  error.value = ''
  try {
    const result = await syncAccountRateRows({ protocol: protocol.value as SupplierChannelCheckProtocol, limit: 1000 })
    rows.value = result.items || []
    appStore.showSuccess(`账号倍率同步完成：匹配 ${result.matched}，失败 ${result.failed + result.not_found + result.ambiguous}`)
  } catch (err) {
    error.value = (err as { message?: string }).message || '账号倍率同步失败'
    appStore.showError(error.value)
  } finally {
    submitting.value = false
  }
}

async function renameAll() {
  if (submitting.value || stats.value.renameable === 0) return
  submitting.value = true
  error.value = ''
  try {
    const result = await renameMatchedAccountRateSyncRows({ protocol: protocol.value as SupplierChannelCheckProtocol, limit: 1000 })
    await loadRows()
    appStore.showSuccess(`账号名称更新完成：${result.renamed}`)
  } catch (err) {
    error.value = (err as { message?: string }).message || '批量更新账号名称失败'
    appStore.showError(error.value)
  } finally {
    submitting.value = false
  }
}

async function retryRow(row: AccountRateSyncRow) {
  if (actionKey.value) return
  const key = `retry:${rowKey(row)}`
  actionKey.value = key
  error.value = ''
  try {
    const next = await retryAccountRateSyncRow(row.local_sub2api_account_id)
    upsertRow(next)
    appStore.showSuccess(next.status === 'matched' || next.status === 'renamed' ? '账号倍率已匹配' : statusLabel(next.status))
  } catch (err) {
    error.value = (err as { message?: string }).message || '重试账号倍率同步失败'
    appStore.showError(error.value)
  } finally {
    if (actionKey.value === key) {
      actionKey.value = ''
    }
  }
}

async function renameRow(row: AccountRateSyncRow) {
  const historyID = row.history?.id
  if (!historyID || actionKey.value) return
  const key = `rename:${rowKey(row)}`
  actionKey.value = key
  error.value = ''
  try {
    const next = await renameAccountRateSyncRow(historyID)
    upsertRow(next)
    appStore.showSuccess('账号名称已更新')
  } catch (err) {
    error.value = (err as { message?: string }).message || '更新账号名称失败'
    appStore.showError(error.value)
  } finally {
    if (actionKey.value === key) {
      actionKey.value = ''
    }
  }
}

async function clearHistory() {
  if (submitting.value) return
  submitting.value = true
  error.value = ''
  try {
    const result = await clearAccountRateSyncHistory()
    await loadRows()
    appStore.showSuccess(`已清空 ${result.deleted} 条同步历史`)
  } catch (err) {
    error.value = (err as { message?: string }).message || '清空同步历史失败'
    appStore.showError(error.value)
  } finally {
    submitting.value = false
  }
}

function upsertRow(next: AccountRateSyncRow) {
  const index = rows.value.findIndex((row) => row.local_sub2api_account_id === next.local_sub2api_account_id)
  if (index >= 0) {
    rows.value = [
      ...rows.value.slice(0, index),
      next,
      ...rows.value.slice(index + 1)
    ]
    return
  }
  rows.value = [next, ...rows.value]
}

function rowKey(row: AccountRateSyncRow): string {
  return String(row.local_sub2api_account_id || row.history?.id || row.local_account_name)
}

function isActionRunning(row: AccountRateSyncRow, action: string): boolean {
  return actionKey.value === `${action}:${rowKey(row)}`
}

function canRename(row: AccountRateSyncRow): boolean {
  return Boolean(row.history?.id && row.target_account_name && row.status === 'matched' && !row.renamed)
}

function statusLabel(status?: AccountRateSyncStatus | string): string {
  if (status === 'matched') return '已匹配'
  if (status === 'renamed') return '已改名'
  if (status === 'ambiguous') return '多重匹配'
  if (status === 'failed') return '失败'
  if (status === 'not_found') return '未匹配'
  return '未同步'
}

function statusClass(status?: AccountRateSyncStatus | string): string {
  if (status === 'matched') return 'badge-primary'
  if (status === 'renamed') return 'badge-success'
  if (status === 'ambiguous') return 'badge-warning'
  if (status === 'failed') return 'badge-danger'
  return 'badge-gray'
}

function protocolLabel(value?: string): string {
  return value === 'anthropic' ? 'Anthropic' : 'OpenAI'
}

function rateText(value?: number): string {
  if (!value || value <= 0) return '-'
  return `${Number(value).toFixed(value >= 0.01 ? 2 : 3).replace(/\.?0+$/, '')}x`
}

function rateClass(row: AccountRateSyncRow): string {
  if (row.effective_rate_multiplier <= 0.1) return 'text-emerald-700 dark:text-emerald-300'
  if (row.effective_rate_multiplier <= 0.5) return 'text-primary-700 dark:text-primary-300'
  return 'text-gray-900 dark:text-gray-100'
}

function errorTitle(row: AccountRateSyncRow): string {
  return [row.error_code, row.error_message].filter(Boolean).join(' · ')
}

watch(protocol, () => {
  void loadRows()
})

onMounted(() => {
  void loadRows()
})
</script>
