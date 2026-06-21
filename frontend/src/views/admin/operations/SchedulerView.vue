<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">调度与插件采集</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            统一管理后端调度、Chrome 插件会话上报和浏览器兜底任务记录。
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadPage">
            <Icon name="refresh" size="sm" />
            刷新
          </button>
          <button type="button" class="btn btn-primary" @click="installDialogOpen = true">
            下载插件
          </button>
        </div>
      </section>

      <section class="grid gap-4 md:grid-cols-5">
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
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">会话上报</p>
          <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">{{ sessionTaskCount }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">执行中</p>
          <p class="mt-2 text-2xl font-semibold text-sky-600 dark:text-sky-400">{{ runningTaskCount }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">失败</p>
          <p class="mt-2 text-2xl font-semibold text-rose-600 dark:text-rose-400">{{ failedTaskCount }}</p>
        </div>
      </section>

      <section class="grid gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
        <form class="card p-5" @submit.prevent="submitRun">
          <div class="flex items-start justify-between gap-3">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">执行调度</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">默认只让插件上报会话；业务同步在后端完成。</p>
            </div>
            <span class="badge badge-gray">{{ status?.queue || 'extension' }}</span>
          </div>

          <div class="mt-5 space-y-4">
            <label class="block">
              <span class="input-label">供应商</span>
              <select v-model.number="form.supplier_id" class="input" @change="handleSupplierChange">
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
                  <span>
                    <span class="block text-gray-700 dark:text-gray-200">{{ option.label }}</span>
                    <span class="text-xs text-gray-500 dark:text-dark-400">{{ option.description }}</span>
                  </span>
                  <input v-model="form.task_types" :value="option.value" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                </label>
              </div>
            </div>

            <button type="button" class="btn btn-secondary w-full" :disabled="diagnosing || form.task_types.length === 0" @click="runDiagnosis">
              {{ diagnosing ? '预检中...' : '预检供应商' }}
            </button>
            <button type="submit" class="btn btn-primary w-full" :disabled="running || form.task_types.length === 0">
              {{ running ? '执行中...' : '执行调度' }}
            </button>
          </div>
        </form>

        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ lastRun ? '执行结果' : '预检结果' }}</h2>
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ runTimeLabel || diagnosisTimeLabel }}</span>
            </div>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full min-w-[960px] divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">供应商</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">任务</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">动作</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">状态</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">任务 ID</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">幂等键</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">结果</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr v-if="visibleRunItems.length === 0">
                  <td colspan="7" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无预检结果</td>
                </tr>
                <tr v-for="item in visibleRunItems" :key="`${item.supplier_id}-${item.task_type}-${item.schedule_key}-${item.reason}`">
                  <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">{{ item.supplier_name || supplierName(item.supplier_id) }}</td>
                  <td class="px-4 py-4"><span class="badge badge-gray">{{ taskTypeLabel(item.task_type) }}</span></td>
                  <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ actionLabel(item.action) }}</td>
                  <td class="px-4 py-4">
                    <span class="badge" :class="runItemClass(item)">{{ runItemStatus(item) }}</span>
                  </td>
                  <td class="px-4 py-4 font-mono text-xs text-gray-500 dark:text-dark-400">{{ item.task_id || '-' }}</td>
                  <td class="px-4 py-4 font-mono text-xs text-gray-500 dark:text-dark-400">{{ item.schedule_key || '-' }}</td>
                  <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ runItemResult(item) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">采集任务记录</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">只展示需要插件执行的会话上报和兼容兜底任务。</p>
            </div>
            <div class="grid gap-2 sm:grid-cols-3">
              <select v-model.number="taskFilters.supplier_id" class="input h-9 py-1 text-sm" @change="resetTaskPagination">
                <option :value="0">全部供应商</option>
                <option v-for="supplier in suppliers" :key="supplier.id" :value="supplier.id">{{ supplier.name }}</option>
              </select>
              <select v-model="taskFilters.status" class="input h-9 py-1 text-sm" @change="resetTaskPagination">
                <option value="">全部状态</option>
                <option value="pending">待领取</option>
                <option value="claimed">已领取</option>
                <option value="running">执行中</option>
                <option value="succeeded">成功</option>
                <option value="failed">失败</option>
              </select>
              <select v-model="taskFilters.type" class="input h-9 py-1 text-sm" @change="resetTaskPagination">
                <option value="">全部类型</option>
                <option v-for="option in taskTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </div>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[1120px] divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">ID</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">供应商</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">类型</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">状态</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">尝试</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">设备</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">结果/错误</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">更新时间</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="tasks.length === 0">
                <td colspan="8" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无采集任务</td>
              </tr>
              <tr v-for="task in tasks" :key="task.id">
                <td class="px-4 py-4 font-mono text-sm text-gray-900 dark:text-gray-100">#{{ task.id }}</td>
                <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">
                  <div>{{ supplierName(task.supplier_id) }}</div>
                  <div v-if="task.schedule_key" class="mt-1 max-w-[220px] truncate font-mono text-xs text-gray-400">{{ task.schedule_key }}</div>
                </td>
                <td class="px-4 py-4"><span class="badge badge-gray">{{ taskTypeLabel(task.type) }}</span></td>
                <td class="px-4 py-4"><span class="badge" :class="taskStatusClass(task.status)">{{ taskStatusLabel(task.status) }}</span></td>
                <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ task.attempts }} / {{ task.max_attempts }}</td>
                <td class="px-4 py-4 font-mono text-xs text-gray-500 dark:text-dark-400">{{ task.device_id || '-' }}</td>
                <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">
                  <button type="button" class="text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="openTaskDetail(task)">
                    {{ task.error_code || resultSummary(task) }}
                  </button>
                </td>
                <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(task.updated_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination
          v-if="taskPagination.total > 0"
          :page="taskPagination.page"
          :total="taskPagination.total"
          :page-size="taskPagination.page_size"
          @update:page="handleTaskPageChange"
          @update:pageSize="handleTaskPageSizeChange"
        />
      </section>

      <div v-if="installDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="installDialogOpen = false">
        <div class="w-full max-w-xl overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-900">
          <div class="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Chrome 插件</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ manifest?.name || 'Sub2API Plus Session Capture' }} {{ manifest?.version ? `v${manifest.version}` : '' }}</p>
            </div>
            <button type="button" class="btn btn-secondary btn-sm" @click="installDialogOpen = false">关闭</button>
          </div>
          <div class="space-y-4 p-5">
            <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">安装流程</p>
              <ol class="mt-3 list-decimal space-y-2 pl-5 text-sm text-gray-600 dark:text-dark-300">
                <li>下载 ZIP 并解压到本地目录。</li>
                <li>打开 <span class="font-mono">chrome://extensions</span>，启用开发者模式。</li>
                <li>选择“加载已解压的扩展程序”，指向解压后的目录。</li>
                <li>在已登录 sub2apiplus 页面连接插件，再到供应商后台一键上报。</li>
              </ol>
            </div>
            <div class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
              <label class="block">
                <span class="input-label">sub2apiplus 地址</span>
                <input :value="adminPlusOrigin" readonly class="input font-mono text-xs" />
              </label>
              <button type="button" class="btn btn-secondary" @click="copyText(adminPlusOrigin, '后台地址已复制')">复制地址</button>
            </div>
            <button type="button" class="btn btn-primary w-full" :disabled="packageDownloading" @click="downloadPackage">
              {{ packageDownloading ? '下载中...' : '下载插件包' }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="detailTask" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="detailTask = null">
        <div class="max-h-[80vh] w-full max-w-3xl overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-900">
          <div class="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">任务 #{{ detailTask.id }}</h3>
            <button type="button" class="btn btn-secondary btn-sm" @click="detailTask = null">关闭</button>
          </div>
          <div class="max-h-[65vh] overflow-auto p-5">
            <pre class="whitespace-pre-wrap rounded-lg bg-gray-50 p-4 text-xs text-gray-700 dark:bg-dark-800 dark:text-gray-200">{{ taskDetailJSON }}</pre>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import {
  downloadExtensionPackage,
  getExtensionManifest,
  getSchedulerStatus,
  listExtensionTasks,
  listSuppliers,
  runScheduler,
  type ExtensionManifestInfo,
  type ExtensionTask,
  type ExtensionTaskType,
  type ScheduledTask,
  type SchedulerRun,
  type SchedulerStatus,
  type Supplier
} from '@/api/admin/adminPlus'

const appStore = useAppStore()

const loading = ref(false)
const running = ref(false)
const diagnosing = ref(false)
const packageDownloading = ref(false)
const installDialogOpen = ref(false)
const suppliers = ref<Supplier[]>([])
const tasks = ref<ExtensionTask[]>([])
const status = ref<SchedulerStatus | null>(null)
const manifest = ref<ExtensionManifestInfo | null>(null)
const lastRun = ref<SchedulerRun | null>(null)
const diagnosis = ref<SchedulerRun | null>(null)
const detailTask = ref<ExtensionTask | null>(null)

const taskPagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const compatTaskTypeOptions: Array<{ value: ExtensionTaskType; label: string; description: string }> = [
  { value: 'fetch_rates', label: '费率同步', description: '后端使用已保存会话读取费率' },
  { value: 'fetch_groups', label: '分组同步', description: '后端使用已保存会话读取分组' },
  { value: 'fetch_balance', label: '余额同步', description: '后端使用已保存会话读取余额' },
  { value: 'fetch_announcements', label: '公告同步', description: '后端使用已保存会话读取供应商公告、通知和充值页' },
  { value: 'fetch_health', label: '健康探测', description: '后端使用本地 Sub2API 账号执行 OpenAI-compatible 探测' },
  { value: 'export_bills', label: '账单同步', description: '后端使用已保存会话读取账单' }
]

const sessionTaskTypeOption = { value: 'capture_supplier_session' as ExtensionTaskType, label: '会话上报', description: '插件只采集并上报供应商浏览器会话' }
const taskTypeOptions = computed(() => [sessionTaskTypeOption, ...compatTaskTypeOptions])

const form = reactive({
  supplier_id: 0,
  window_minutes: 10,
  task_types: ['capture_supplier_session'] as ExtensionTaskType[]
})

const taskFilters = reactive({
  supplier_id: 0,
  status: '',
  type: ''
})

const adminPlusOrigin = computed(() => (typeof window === 'undefined' ? '' : window.location.origin))
const intervalLabel = computed(() => {
  const seconds = status.value?.interval_seconds || 0
  if (seconds <= 0) return '-'
  if (seconds % 60 === 0) return `${seconds / 60} 分钟`
  return `${seconds} 秒`
})
const runTimeLabel = computed(() => formatDateTime(lastRun.value?.requested_at))
const diagnosisTimeLabel = computed(() => formatDateTime(diagnosis.value?.requested_at))
const visibleRunItems = computed(() => lastRun.value?.items || diagnosis.value?.items || [])
const runningTaskCount = computed(() => tasks.value.filter((task) => ['claimed', 'running'].includes(task.status)).length)
const failedTaskCount = computed(() => tasks.value.filter((task) => task.status === 'failed').length)
const sessionTaskCount = computed(() => tasks.value.filter((task) => task.type === 'capture_supplier_session').length)
const taskDetailJSON = computed(() => JSON.stringify(detailTask.value, null, 2))

function supplierName(id: number): string {
  return suppliers.value.find((supplier) => supplier.id === id)?.name || `#${id}`
}

function taskTypeLabel(value: ExtensionTaskType): string {
  return taskTypeOptions.value.find((option) => option.value === value)?.label || value
}

function reasonLabel(reason?: string): string {
  if (!reason) return '-'
  return {
    duplicate: '同一窗口已存在任务',
    supplier_disabled: '供应商已停用',
    supplier_paused: '供应商已暂停',
    credential_invalid: '凭据失效',
    supplier_url_missing: '缺少后台或 API 地址',
    browser_login_disabled: '未启用 Chrome 登录',
    dashboard_url_missing: '缺少后台地址',
    browser_login_credential_missing: '缺少登录账号或 Token',
    not_switch_eligible: '无可用余额或不可切换',
    group_syncer_missing: '分组同步未配置',
    rate_syncer_missing: '费率同步未配置',
    balance_syncer_missing: '余额同步未配置',
    announcement_syncer_missing: '公告同步未配置',
    health_syncer_missing: '健康探测未配置',
    billing_syncer_missing: '账单同步未配置',
    direct_sync_not_supported: '不支持后端直连'
  }[reason] || reason
}

function actionLabel(action?: ScheduledTask['action']): string {
  return {
    direct_sync: '后端同步',
    extension_task: '插件任务',
    compat_task: '兼容兜底'
  }[action || 'compat_task']
}

function runItemStatus(item: ScheduledTask): string {
  if (item.synced) return '已同步'
  if (item.created) return '已创建'
  if (item.reason) return '已跳过'
  return item.action === 'direct_sync' ? '可同步' : '可创建'
}

function runItemClass(item: ScheduledTask): string {
  if (item.synced) return 'badge-success'
  if (item.created) return 'badge-success'
  if (item.reason) return 'badge-warning'
  return 'badge-success'
}

function runItemResult(item: ScheduledTask): string {
  if (item.reason) return reasonLabel(item.reason)
  if (item.synced) return `同步 ${item.total || 0} 条`
  if (item.created) return '已写入插件队列'
  return item.action === 'direct_sync' ? '预检通过' : '可写入插件队列'
}

function taskStatusLabel(statusValue: ExtensionTask['status']): string {
  return {
    pending: '待领取',
    claimed: '已领取',
    running: '执行中',
    succeeded: '成功',
    failed: '失败',
    cancelled: '已取消'
  }[statusValue] || statusValue
}

function taskStatusClass(statusValue: ExtensionTask['status']): string {
  if (statusValue === 'succeeded') return 'badge-success'
  if (statusValue === 'failed' || statusValue === 'cancelled') return 'badge-danger'
  if (['pending', 'claimed', 'running'].includes(statusValue)) return 'badge-warning'
  return 'badge-gray'
}

function formatDateTime(value?: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleString()
}

function resultSummary(task: ExtensionTask): string {
  if (task.status === 'succeeded') {
    const summary = task.result?.session_summary as Record<string, unknown> | undefined
    if (summary) return `会话: ${summary.cookie_count || 0} cookies`
    const ingest = task.result?.ingest as Record<string, unknown> | undefined
    if (ingest) return Object.entries(ingest).map(([key, value]) => `${key}:${value}`).join(' / ')
    if (task.result) return '查看结果'
  }
  if (task.status === 'failed') return task.error_message || '查看错误'
  return '查看详情'
}

function openTaskDetail(task: ExtensionTask) {
  detailTask.value = task
}

async function copyText(value: string, message: string) {
  await navigator.clipboard.writeText(value)
  appStore.showSuccess(message)
}

async function loadPage() {
  loading.value = true
  try {
    const [supplierResult, schedulerStatus, extensionManifest] = await Promise.all([
      listSuppliers(),
      getSchedulerStatus(),
      getExtensionManifest()
    ])
    suppliers.value = supplierResult.items
    status.value = schedulerStatus
    manifest.value = extensionManifest
    await Promise.all([runDiagnosis(), loadTasks()])
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '加载调度与插件采集失败')
  } finally {
    loading.value = false
  }
}

async function runDiagnosis() {
  diagnosing.value = true
  try {
    diagnosis.value = await runScheduler({
      mode: 'diagnose',
      supplier_id: form.supplier_id || undefined,
      task_types: form.task_types,
      window_minutes: Number(form.window_minutes || 10),
      dry_run: true
    })
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '预检供应商失败')
  } finally {
    diagnosing.value = false
  }
}

async function loadTasks() {
  const taskResult = await listExtensionTasks({
    supplier_id: taskFilters.supplier_id || form.supplier_id || undefined,
    status: taskFilters.status || undefined,
    type: taskFilters.type || undefined,
    page: taskPagination.page,
    page_size: taskPagination.page_size
  })
  tasks.value = taskResult.items
  taskPagination.total = taskResult.total || 0
  taskPagination.pages = taskResult.pages || 0
  taskPagination.page = taskResult.page || taskPagination.page
  taskPagination.page_size = taskResult.page_size || taskPagination.page_size
}

function handleSupplierChange() {
  taskFilters.supplier_id = form.supplier_id
  resetTaskPagination()
  void runDiagnosis()
}

function resetTaskPagination() {
  taskPagination.page = 1
  void loadTasks()
}

function handleTaskPageChange(page: number) {
  taskPagination.page = page
  void loadTasks()
}

function handleTaskPageSizeChange(pageSize: number) {
  taskPagination.page_size = pageSize
  taskPagination.page = 1
  void loadTasks()
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
    const syncedCount = lastRun.value.items.filter((item) => item.synced).length
    appStore.showSuccess(`已同步 ${syncedCount} 项，创建 ${lastRun.value.created_count} 个插件任务，跳过 ${lastRun.value.skipped_count} 项`)
    await Promise.all([runDiagnosis(), loadTasks()])
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '执行调度失败')
  } finally {
    running.value = false
  }
}

async function downloadPackage() {
  packageDownloading.value = true
  try {
    const blob = await downloadExtensionPackage(adminPlusOrigin.value)
    const version = manifest.value?.version || 'dev'
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `sub2api-plus-session-capture-${version}.zip`
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    URL.revokeObjectURL(url)
    appStore.showSuccess('插件包已下载')
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '下载插件包失败')
  } finally {
    packageDownloading.value = false
  }
}

onMounted(loadPage)
</script>
