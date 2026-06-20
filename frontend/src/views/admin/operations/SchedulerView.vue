<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">调度中心</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            根据供应商状态生成 Chrome 插件采集任务，进入插件任务队列后由插件领取执行。
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadPage">
            <Icon name="refresh" size="sm" />
            刷新
          </button>
          <RouterLink class="btn btn-secondary" to="/admin/operations/extension-tasks">
            查看插件任务
          </RouterLink>
        </div>
      </section>

      <section class="grid gap-4 md:grid-cols-4">
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">周期调度</p>
          <p class="mt-2 text-2xl font-semibold" :class="status?.enabled ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-900 dark:text-white'">
            {{ status?.enabled ? '已开启' : '已关闭' }}
          </p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">间隔</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ intervalLabel }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">本次创建</p>
          <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">{{ lastRun?.created_count ?? 0 }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">本次跳过</p>
          <p class="mt-2 text-2xl font-semibold text-amber-600 dark:text-amber-400">{{ lastRun?.skipped_count ?? 0 }}</p>
        </div>
      </section>

      <section class="grid gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
        <form class="card p-5" @submit.prevent="submitRun">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">手动生成任务</h2>
          <div class="mt-5 space-y-4">
            <label class="block">
              <span class="input-label">供应商</span>
              <select v-model.number="form.supplier_id" class="input">
                <option :value="0">全部供应商</option>
                <option v-for="supplier in suppliers" :key="supplier.id" :value="supplier.id">
                  {{ supplier.name }}
                </option>
              </select>
            </label>

            <label class="block">
              <span class="input-label">窗口分钟</span>
              <input v-model.number="form.window_minutes" type="number" min="1" max="1440" class="input" />
            </label>

            <div>
              <span class="input-label">任务类型</span>
              <div class="mt-2 grid gap-2">
                <label v-for="option in taskTypeOptions" :key="option.value" class="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 text-sm dark:border-dark-700">
                  <span class="text-gray-700 dark:text-gray-200">{{ option.label }}</span>
                  <input v-model="form.task_types" :value="option.value" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                </label>
              </div>
            </div>

            <button type="submit" class="btn btn-primary w-full" :disabled="running || form.task_types.length === 0">
              {{ running ? '生成中...' : '生成插件任务' }}
            </button>
          </div>
        </form>

        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">生成结果</h2>
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ runTimeLabel }}</span>
            </div>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full min-w-[960px] divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">供应商</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">任务</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">状态</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">任务 ID</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">幂等键</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">原因</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr v-if="!lastRun">
                  <td colspan="6" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">尚未执行手动调度</td>
                </tr>
                <tr v-for="item in lastRun?.items || []" :key="`${item.supplier_id}-${item.task_type}-${item.schedule_key}`">
                  <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">{{ item.supplier_name || supplierName(item.supplier_id) }}</td>
                  <td class="px-4 py-4"><span class="badge badge-gray">{{ taskTypeLabel(item.task_type) }}</span></td>
                  <td class="px-4 py-4">
                    <span class="badge" :class="item.created ? 'badge-success' : 'badge-warning'">{{ item.created ? '已创建' : '已跳过' }}</span>
                  </td>
                  <td class="px-4 py-4 font-mono text-xs text-gray-500 dark:text-dark-400">{{ item.task_id || '-' }}</td>
                  <td class="px-4 py-4 font-mono text-xs text-gray-500 dark:text-dark-400">{{ item.schedule_key || '-' }}</td>
                  <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ reasonLabel(item.reason) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import {
  getSchedulerStatus,
  listSuppliers,
  runScheduler,
  type ExtensionTaskType,
  type SchedulerRun,
  type SchedulerStatus,
  type Supplier
} from '@/api/admin/adminPlus'

const appStore = useAppStore()

const loading = ref(false)
const running = ref(false)
const suppliers = ref<Supplier[]>([])
const status = ref<SchedulerStatus | null>(null)
const lastRun = ref<SchedulerRun | null>(null)

const taskTypeOptions: Array<{ value: ExtensionTaskType; label: string }> = [
  { value: 'fetch_rates', label: '抓取费率' },
  { value: 'fetch_balance', label: '抓取余额' },
  { value: 'fetch_promotions', label: '抓取优惠' },
  { value: 'fetch_health', label: '抓取健康' },
  { value: 'export_bills', label: '导出账单' }
]

const form = reactive({
  supplier_id: 0,
  window_minutes: 10,
  task_types: ['fetch_rates', 'fetch_balance', 'fetch_promotions', 'fetch_health', 'export_bills'] as ExtensionTaskType[]
})

const intervalLabel = computed(() => {
  const seconds = status.value?.interval_seconds || 0
  if (seconds <= 0) return '-'
  if (seconds % 60 === 0) return `${seconds / 60} 分钟`
  return `${seconds} 秒`
})

const runTimeLabel = computed(() => {
  if (!lastRun.value?.requested_at) return ''
  const date = new Date(lastRun.value.requested_at)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleString()
})

function supplierName(id: number): string {
  return suppliers.value.find((supplier) => supplier.id === id)?.name || `#${id}`
}

function taskTypeLabel(value: ExtensionTaskType): string {
  return taskTypeOptions.find((option) => option.value === value)?.label || value
}

function reasonLabel(reason?: string): string {
  if (!reason) return '-'
  return {
    duplicate: '同一窗口已存在任务',
    supplier_disabled: '供应商已停用',
    supplier_paused: '供应商已暂停',
    credential_invalid: '凭据失效',
    browser_login_disabled: '未启用 Chrome 登录',
    dashboard_url_missing: '缺少后台地址',
    browser_login_credential_missing: '缺少登录账号或 Token',
    not_switch_eligible: '无可用余额或不可切换'
  }[reason] || reason
}

async function loadPage() {
  loading.value = true
  try {
    const [supplierResult, schedulerStatus] = await Promise.all([
      listSuppliers(),
      getSchedulerStatus()
    ])
    suppliers.value = supplierResult.items
    status.value = schedulerStatus
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '加载调度中心失败')
  } finally {
    loading.value = false
  }
}

async function submitRun() {
  running.value = true
  try {
    lastRun.value = await runScheduler({
      mode: 'manual',
      supplier_id: form.supplier_id || undefined,
      task_types: form.task_types,
      window_minutes: Number(form.window_minutes || 10)
    })
    appStore.showSuccess(`已创建 ${lastRun.value.created_count} 个任务，跳过 ${lastRun.value.skipped_count} 个`)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '生成插件任务失败')
  } finally {
    running.value = false
  }
}

onMounted(loadPage)
</script>
