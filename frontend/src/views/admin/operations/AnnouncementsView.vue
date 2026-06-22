<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">公告监控</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">读取供应商公告、通知和充值页，并按关键词分类生成运营信号。</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <select v-model.number="selectedSupplierId" class="input w-64 max-w-full">
            <option :value="0">全部供应商</option>
            <option v-for="supplier in suppliers" :key="supplier.id" :value="supplier.id">
              {{ supplier.name }}
            </option>
          </select>
          <button type="button" class="btn btn-primary" :disabled="!selectedSupplierId || syncing" @click="syncSelectedSupplier">
            <Icon name="sync" size="sm" />
            {{ syncing ? '同步中...' : '同步公告' }}
          </button>
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadPage">
            <Icon name="refresh" size="sm" />
            刷新
          </button>
        </div>
      </section>

      <section class="grid gap-4 md:grid-cols-4">
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">公告事件</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ eventPagination.total }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">可切换</p>
          <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">{{ switchCandidateCount }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">需充值</p>
          <p class="mt-2 text-2xl font-semibold text-amber-600 dark:text-amber-400">{{ rechargeUnlockCount }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">待处理</p>
          <p class="mt-2 text-2xl font-semibold text-rose-600 dark:text-rose-400">{{ openCount }}</p>
        </div>
      </section>

      <section class="card overflow-hidden">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">公告事件</h2>
          <div class="flex flex-wrap items-center gap-2">
            <select v-model="filters.recommendation" class="input w-40" @change="resetAndLoad">
              <option value="">全部建议</option>
              <option value="switch_candidate">可切换</option>
              <option value="recharge_to_unlock">需充值</option>
              <option value="monitor_only">仅监控</option>
              <option value="informational">信息</option>
            </select>
            <select v-model="filters.status" class="input w-36" @change="resetAndLoad">
              <option value="">全部状态</option>
              <option value="open">待处理</option>
              <option value="acknowledged">已确认</option>
              <option value="ignored">已忽略</option>
            </select>
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900/40">
              <tr>
                <th class="table-th">供应商</th>
                <th class="table-th">公告</th>
                <th class="table-th">建议</th>
                <th class="table-th">余额</th>
                <th class="table-th">门槛</th>
                <th class="table-th">比例</th>
                <th class="table-th">来源</th>
                <th class="table-th">时间</th>
                <th class="table-th text-right">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-if="events.length === 0">
                <td colspan="9" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无公告事件</td>
              </tr>
              <tr v-for="event in events" :key="event.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="table-td">
                  <div class="font-medium text-gray-900 dark:text-white">{{ supplierName(event.supplier_id) }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">#{{ event.supplier_id }}</div>
                </td>
                <td class="table-td min-w-[280px]">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="badge badge-gray">{{ typeText(event.type) }}</span>
                    <span class="badge" :class="event.status === 'open' ? 'badge-warning' : 'badge-success'">{{ statusText(event.status) }}</span>
                  </div>
                  <div class="mt-2 font-medium text-gray-900 dark:text-white">{{ event.title }}</div>
                  <div class="mt-1 max-w-xl text-sm text-gray-500 dark:text-dark-400">{{ event.description || '-' }}</div>
                </td>
                <td class="table-td">
                  <span class="badge" :class="recommendationClass(event.recommendation)">{{ recommendationText(event.recommendation) }}</span>
                </td>
                <td class="table-td whitespace-nowrap">{{ formatMoney(event.balance_cents, event.currency) }}</td>
                <td class="table-td whitespace-nowrap">{{ formatMoney(event.min_recharge_cents, event.currency) }}</td>
                <td class="table-td whitespace-nowrap">
                  <div v-if="event.bonus_percent">赠送 {{ formatPercent(event.bonus_percent) }}</div>
                  <div v-if="event.discount_percent">折扣 {{ formatPercent(event.discount_percent) }}</div>
                  <span v-if="!event.bonus_percent && !event.discount_percent">-</span>
                </td>
                <td class="table-td whitespace-nowrap">{{ sourceText(event.source) }}</td>
                <td class="table-td whitespace-nowrap">{{ formatDate(event.captured_at) }}</td>
                <td class="table-td text-right">
                  <button v-if="event.status === 'open'" type="button" class="btn btn-secondary px-3 py-1.5 text-xs" @click="ackEvent(event.id)">
                    确认
                  </button>
                  <span v-else class="text-xs text-gray-400">-</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="eventPagination.total > 0"
          :page="eventPagination.page"
          :total="eventPagination.total"
          :page-size="eventPagination.page_size"
          @update:page="handleEventPageChange"
          @update:pageSize="handleEventPageSizeChange"
        />
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import {
  acknowledgeAnnouncementEvent,
  listAnnouncementEvents,
  listSuppliers,
  syncSupplierAnnouncements,
  type AnnouncementEvent,
  type Supplier
} from '@/api/admin/adminPlus'

const appStore = useAppStore()

const loading = ref(false)
const syncing = ref(false)
const suppliers = ref<Supplier[]>([])
const events = ref<AnnouncementEvent[]>([])
const selectedSupplierId = ref(0)
const filters = reactive({
  status: '',
  recommendation: ''
})
const eventPagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0, pages: 0 })

const switchCandidateCount = computed(() => events.value.filter((event) => event.recommendation === 'switch_candidate').length)
const rechargeUnlockCount = computed(() => events.value.filter((event) => event.recommendation === 'recharge_to_unlock').length)
const openCount = computed(() => events.value.filter((event) => event.status === 'open').length)

function formatMoney(cents: number, currency: string): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: currency || 'USD',
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: 2
  }).format((cents || 0) / 100)
}

function formatPercent(value: number): string {
  return `${Number(value || 0).toFixed(2)}%`
}

function formatDate(value: string): string {
  if (!value) return '-'
  return new Intl.DateTimeFormat(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
}

function supplierName(id: number): string {
  return suppliers.value.find((supplier) => supplier.id === id)?.name || `#${id}`
}

function typeText(type: AnnouncementEvent['type']): string {
  if (type === 'recharge_bonus') return '充值赠送'
  if (type === 'rate_discount') return '费率折扣'
  if (type === 'package_deal') return '套餐折扣'
  if (type === 'limited_offer') return '限时活动'
  if (type === 'maintenance') return '维护公告'
  if (type === 'incident') return '故障公告'
  if (type === 'notice') return '普通通知'
  return '其他'
}

function recommendationText(recommendation: AnnouncementEvent['recommendation']): string {
  if (recommendation === 'switch_candidate') return '可切换'
  if (recommendation === 'recharge_to_unlock') return '需充值'
  if (recommendation === 'monitor_only') return '仅监控'
  return '信息'
}

function recommendationClass(recommendation: AnnouncementEvent['recommendation']): string {
  if (recommendation === 'switch_candidate') return 'badge-success'
  if (recommendation === 'recharge_to_unlock') return 'badge-warning'
  return 'badge-gray'
}

function statusText(status: AnnouncementEvent['status']): string {
  if (status === 'open') return '待处理'
  if (status === 'acknowledged') return '已确认'
  if (status === 'ignored') return '已忽略'
  return status
}

function sourceText(source: string): string {
  if (source === 'provider_session') return '供应商会话'
  if (source === 'manual') return '手工'
  return source || '-'
}

async function loadPage() {
  loading.value = true
  try {
    const [supplierResult, eventResult] = await Promise.all([
      listSuppliers(),
      listAnnouncementEvents({
        page: eventPagination.page,
        page_size: eventPagination.page_size,
        supplier_id: selectedSupplierId.value || undefined,
        status: filters.status || undefined,
        recommendation: filters.recommendation || undefined
      })
    ])
    suppliers.value = supplierResult.items
    events.value = eventResult.items
    eventPagination.total = eventResult.total || 0
    eventPagination.pages = eventResult.pages || 0
    eventPagination.page = eventResult.page || eventPagination.page
    eventPagination.page_size = eventResult.page_size || eventPagination.page_size
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '加载公告事件失败')
  } finally {
    loading.value = false
  }
}

async function syncSelectedSupplier() {
  if (!selectedSupplierId.value) return
  syncing.value = true
  try {
    const result = await syncSupplierAnnouncements(selectedSupplierId.value)
    appStore.showSuccess(result.total > 0 ? `已同步 ${result.total} 条公告事件` : '未发现新的公告事件')
    eventPagination.page = 1
    await loadPage()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '同步公告失败')
  } finally {
    syncing.value = false
  }
}

function resetAndLoad() {
  eventPagination.page = 1
  void loadPage()
}

function handleEventPageChange(page: number) {
  eventPagination.page = page
  void loadPage()
}

function handleEventPageSizeChange(pageSize: number) {
  eventPagination.page_size = pageSize
  eventPagination.page = 1
  void loadPage()
}

async function ackEvent(id: number) {
  try {
    await acknowledgeAnnouncementEvent(id)
    appStore.showSuccess('事件已确认')
    await loadPage()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '确认事件失败')
  }
}

watch(selectedSupplierId, () => {
  resetAndLoad()
})

onMounted(loadPage)
</script>
