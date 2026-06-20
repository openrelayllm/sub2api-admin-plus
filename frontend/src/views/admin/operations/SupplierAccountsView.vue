<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap-reverse items-start justify-between gap-3">
          <div class="grid flex-1 gap-3 lg:grid-cols-[260px_minmax(220px,1fr)_160px_160px]">
            <label class="block">
              <span class="input-label">供应商</span>
              <select v-model.number="selectedSupplierID" class="input">
                <option :value="0">全部供应商</option>
                <option v-for="supplier in suppliers" :key="supplier.id" :value="supplier.id">
                  {{ supplier.name }} · {{ typeLabel(supplier.type) }}
                </option>
              </select>
            </label>
            <label class="block">
              <span class="input-label">搜索</span>
              <div class="relative">
                <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input v-model.trim="filters.q" class="input pl-9" placeholder="本地账号、供应商侧标识、标签、费率档案" />
              </div>
            </label>
            <label class="block">
              <span class="input-label">运行状态</span>
              <select v-model="filters.runtime_status" class="input">
                <option value="">全部</option>
                <option value="monitor_only">仅监控</option>
                <option value="candidate">候选</option>
                <option value="active">当前使用</option>
                <option value="disabled">停用</option>
              </select>
            </label>
            <label class="block">
              <span class="input-label">健康状态</span>
              <select v-model="filters.health_status" class="input">
                <option value="">全部</option>
                <option value="normal">正常</option>
                <option value="unavailable">不可用</option>
                <option value="credential_invalid">凭据失效</option>
                <option value="paused">暂停</option>
              </select>
            </label>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <button type="button" class="btn btn-secondary px-2 md:px-3" :disabled="loading" title="刷新" @click="loadAll">
              <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
              <span class="hidden md:inline">刷新</span>
            </button>
            <button type="button" class="btn btn-secondary px-2 md:px-3" title="清除筛选" @click="resetFilters">
              <Icon name="x" size="sm" />
              <span class="hidden md:inline">清除筛选</span>
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="pagedBindings"
          :loading="loadingBindings"
          row-key="id"
          default-sort-key="id"
          default-sort-order="desc"
          :estimate-row-height="76"
        >
          <template #cell-supplier="{ row }">
            <div class="min-w-[180px]">
              <div class="font-medium text-gray-900 dark:text-white">{{ supplierLabel(row.supplier_id) }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">父级 #{{ row.supplier_id }}</div>
            </div>
          </template>

          <template #cell-local_account="{ row }">
            <div class="min-w-[220px]">
              <div class="font-medium text-gray-900 dark:text-white">{{ row.local_account_name }}</div>
              <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                <span class="font-mono">#{{ row.local_sub2api_account_id }}</span>
                <span>{{ row.local_account_platform }} / {{ row.local_account_type }}</span>
              </div>
            </div>
          </template>

          <template #cell-supplier_account="{ row }">
            <div class="min-w-[220px]">
              <div class="text-sm text-gray-900 dark:text-gray-100">{{ row.supplier_account_label || '-' }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ row.supplier_account_identifier || '-' }}</div>
              <div v-if="row.organization_id || row.project_id" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ row.organization_id || '-' }} / {{ row.project_id || '-' }}
              </div>
            </div>
          </template>

          <template #cell-status="{ row }">
            <div class="flex min-w-[150px] flex-col gap-1.5">
              <div class="flex flex-wrap gap-1.5">
                <span class="badge w-fit" :class="runtimeClass(row.runtime_status)">{{ runtimeLabel(row.runtime_status) }}</span>
                <span class="badge w-fit" :class="healthClass(row.health_status)">{{ healthLabel(row.health_status) }}</span>
              </div>
              <span class="text-xs font-medium" :class="switchStateClass(row)">
                {{ switchStateLabel(row) }}
              </span>
            </div>
          </template>

          <template #cell-balance="{ row }">
            <div class="min-w-[150px] text-right">
              <div class="text-base font-semibold" :class="balanceAmountClass(row)">
                {{ formatMoney(row.balance_cents, row.balance_currency) }}
              </div>
              <div class="text-xs text-gray-500 dark:text-dark-400">阈值 {{ formatMoney(row.balance_threshold_cents, row.balance_currency) }}</div>
              <span v-if="row.has_usable_balance" class="badge badge-success mt-1">有额度</span>
              <span v-else class="badge badge-gray mt-1">无额度</span>
            </div>
          </template>

          <template #cell-concurrency="{ row }">
            <div class="min-w-[110px] text-right">
              <div>{{ row.observed_max_concurrency || 0 }} / {{ row.configured_concurrency || 0 }}</div>
              <div class="text-xs text-gray-500 dark:text-dark-400">观测 / 配置</div>
            </div>
          </template>

          <template #cell-rate_profile="{ row }">
            <div class="min-w-[170px]">
              <div class="text-base font-semibold text-primary-700 dark:text-primary-300">
                {{ rateProfileLabel(row.rate_profile) }}
              </div>
              <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">成本核算优先字段</div>
            </div>
          </template>

          <template #cell-created_at="{ row }">
            <div class="min-w-[150px] text-xs text-gray-500 dark:text-dark-400">{{ formatDateTime(row.created_at) }}</div>
          </template>

          <template #empty>
            <EmptyState
              title="暂无账号/Key 绑定"
              description="请在供应商管理页打开分组弹窗，同步分组后从分组行开通 Key/账号；这里仅展示已生成的绑定。"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import {
  listSupplierAccounts,
  listSuppliers,
  type Supplier,
  type SupplierAccount,
  type SupplierHealthStatus,
  type SupplierRuntimeStatus,
  type SupplierType
} from '@/api/admin/adminPlus'

const appStore = useAppStore()
const route = useRoute()

const loading = ref(false)
const loadingBindings = ref(false)
const suppliers = ref<Supplier[]>([])
const bindings = ref<SupplierAccount[]>([])
const selectedSupplierID = ref(0)

const filters = reactive({
  q: '',
  runtime_status: '' as '' | SupplierRuntimeStatus,
  health_status: '' as '' | SupplierHealthStatus
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const columns: Column[] = [
  { key: 'supplier', label: '供应商父级' },
  { key: 'rate_profile', label: '费率', class: 'font-semibold' },
  { key: 'balance', label: '余额', class: 'text-right' },
  { key: 'status', label: '状态' },
  { key: 'local_account', label: '本地 Sub2API 账号', sortable: true },
  { key: 'supplier_account', label: '供应商侧账号/Key' },
  { key: 'concurrency', label: '并发', class: 'text-right' },
  { key: 'created_at', label: '创建时间', sortable: true }
]

const filteredBindings = computed(() => {
  const q = filters.q.toLowerCase()
  return bindings.value.filter((item) => {
    if (filters.runtime_status && item.runtime_status !== filters.runtime_status) return false
    if (filters.health_status && item.health_status !== filters.health_status) return false
    if (q) {
      const haystack = [
        item.local_account_name,
        item.local_account_platform,
        item.local_account_type,
        item.supplier_account_identifier || '',
        item.supplier_account_label || '',
        item.organization_id || '',
        item.project_id || '',
        item.rate_profile || ''
      ].join(' ').toLowerCase()
      if (!haystack.includes(q)) return false
    }
    return true
  })
})

const pagedBindings = computed(() => {
  const start = (pagination.page - 1) * pagination.page_size
  return filteredBindings.value.slice(start, start + pagination.page_size)
})

function formatMoney(cents: number, currency: string): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: currency || 'CNY',
    minimumFractionDigits: 2
  }).format((cents || 0) / 100)
}

function formatDateTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString()
}

function typeLabel(value: SupplierType): string {
  return {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    gemini: 'Gemini',
    sub2api: 'Sub2API',
    new_api: 'New API',
    browser_only: '仅浏览器',
    custom: '自定义'
  }[value]
}

function runtimeLabel(value: SupplierRuntimeStatus): string {
  return {
    monitor_only: '仅监控',
    candidate: '候选',
    active: '使用中',
    disabled: '停用'
  }[value]
}

function healthLabel(value: SupplierHealthStatus): string {
  return {
    normal: '正常',
    unavailable: '不可用',
    credential_invalid: '凭据失效',
    paused: '暂停'
  }[value]
}

function runtimeClass(status: SupplierRuntimeStatus): string {
  if (status === 'active') return 'badge-success'
  if (status === 'candidate') return 'badge-primary'
  if (status === 'disabled') return 'badge-danger'
  return 'badge-gray'
}

function healthClass(status: SupplierHealthStatus): string {
  if (status === 'normal') return 'badge-success'
  if (status === 'paused') return 'badge-warning'
  return 'badge-danger'
}

function rateProfileLabel(value?: string): string {
  return value?.trim() || 'default'
}

function balanceAmountClass(row: SupplierAccount): string {
  if (row.has_usable_balance) return 'text-emerald-700 dark:text-emerald-300'
  if ((row.balance_cents || 0) <= 0) return 'text-red-600 dark:text-red-300'
  return 'text-gray-900 dark:text-gray-100'
}

function switchStateLabel(row: SupplierAccount): string {
  if (row.runtime_status === 'active' && row.health_status === 'normal' && row.has_usable_balance) return '当前承载流量'
  if (row.runtime_status === 'candidate' && row.health_status === 'normal' && row.has_usable_balance) return '可进入候选'
  if (row.runtime_status === 'monitor_only') return '仅监控，不切换'
  if (!row.has_usable_balance) return '余额不足，不切换'
  if (row.health_status !== 'normal') return '健康异常，不切换'
  return '不可切换'
}

function switchStateClass(row: SupplierAccount): string {
  if ((row.runtime_status === 'active' || row.runtime_status === 'candidate') && row.health_status === 'normal' && row.has_usable_balance) {
    return 'text-emerald-700 dark:text-emerald-300'
  }
  if (!row.has_usable_balance || row.health_status !== 'normal') {
    return 'text-red-600 dark:text-red-300'
  }
  return 'text-gray-500 dark:text-dark-400'
}

function supplierLabel(id: number): string {
  return suppliers.value.find((supplier) => supplier.id === id)?.name || `#${id}`
}

async function loadSuppliers() {
  const result = await listSuppliers()
  suppliers.value = result.items
  const querySupplierID = Number(route.query.supplier_id || 0)
  if (querySupplierID && suppliers.value.some((supplier) => supplier.id === querySupplierID)) {
    selectedSupplierID.value = querySupplierID
  } else if (!selectedSupplierID.value && suppliers.value.length > 0) {
    selectedSupplierID.value = suppliers.value[0].id
  }
}

async function loadBindings() {
  loadingBindings.value = true
  try {
    if (selectedSupplierID.value) {
      const result = await listSupplierAccounts(selectedSupplierID.value, { page: 1, page_size: 1000 })
      bindings.value = result.items
      syncBindingPagination()
      return
    }
    const all: SupplierAccount[] = []
    for (const supplier of suppliers.value) {
      const result = await listSupplierAccounts(supplier.id, { page: 1, page_size: 1000 })
      all.push(...result.items)
    }
    bindings.value = all
    syncBindingPagination()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '加载账号/Key 绑定失败')
  } finally {
    loadingBindings.value = false
  }
}

function syncBindingPagination() {
  pagination.total = filteredBindings.value.length
  pagination.pages = Math.ceil(pagination.total / pagination.page_size)
  if (pagination.page > Math.max(1, pagination.pages)) {
    pagination.page = Math.max(1, pagination.pages)
  }
}

function reloadFirstPage() {
  pagination.page = 1
  void loadBindings()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadBindings()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadBindings()
}

async function loadAll() {
  loading.value = true
  try {
    await loadSuppliers()
    await loadBindings()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '加载数据失败')
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  filters.q = ''
  filters.runtime_status = ''
  filters.health_status = ''
  reloadFirstPage()
}

watch(selectedSupplierID, () => {
  reloadFirstPage()
})

watch(
  () => ({ ...filters }),
  () => {
    reloadFirstPage()
  }
)

onMounted(loadAll)
</script>
