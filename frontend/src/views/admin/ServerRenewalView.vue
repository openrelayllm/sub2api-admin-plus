<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">续费提醒</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            提醒服务器即将到期，帮助及时去服务商面板完成续费。
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadPage">
            <Icon name="refresh" size="sm" />
            刷新
          </button>
          <button type="button" class="btn btn-primary" :disabled="saving" @click="saveRenewal">
            <Icon name="check" size="sm" />
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </section>

      <nav class="flex gap-2 overflow-x-auto border-b border-gray-200 dark:border-dark-700">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          type="button"
          class="whitespace-nowrap border-b-2 px-3 py-2 text-sm font-medium"
          :class="activeTab === tab.value ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-900 dark:text-dark-400 dark:hover:text-white'"
          @click="activeTab = tab.value"
        >
          {{ tab.label }}
        </button>
      </nav>

      <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">提醒开关</p>
          <p class="mt-2 text-2xl font-semibold" :class="form.enabled ? 'text-emerald-700 dark:text-emerald-400' : 'text-gray-700 dark:text-dark-200'">
            {{ form.enabled ? '已开启' : '已关闭' }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ form.server_name || '未命名服务器' }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">到期剩余</p>
          <p class="mt-2 text-2xl font-semibold" :class="stateTextClass">{{ daysLabel }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ statusLabel(status?.state) }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">到期时间</p>
          <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ expiryDisplay }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ form.provider || '未知服务商' }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">下次提醒</p>
          <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ status?.next_reminder || '-' }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ reminderDaysText || '7,3,1' }} 天</p>
        </div>
      </section>

      <section v-if="activeTab === 'dashboard'" class="grid gap-6 xl:grid-cols-[minmax(0,1.6fr)_minmax(320px,0.8fr)]">
        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">续费工作台</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">只处理服务器到期提醒，不记录登录凭据或执行服务器操作。</p>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div class="grid gap-4 px-5 py-4 lg:grid-cols-[160px_minmax(0,1fr)_180px] lg:items-center">
              <div>
                <span class="badge" :class="statusBadgeClass(status?.state)">{{ statusLabel(status?.state) }}</span>
                <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ form.enabled ? '提醒已启用' : '提醒已关闭' }}</p>
              </div>
              <div>
                <p class="font-medium text-gray-900 dark:text-white">{{ form.server_name || '未命名服务器' }}</p>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ form.provider || '未知服务商' }} · {{ form.host_id ? `Host ID ${form.host_id}` : 'Host ID 未配置' }}
                </p>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  到期时间：{{ expiryDisplay }} · 剩余：{{ daysLabel }}
                </p>
              </div>
              <div class="flex justify-start lg:justify-end">
                <button type="button" class="btn btn-primary btn-sm" :disabled="!form.panel_url" @click="openRenewalPanel">
                  <Icon name="externalLink" size="xs" />
                  去续费
                </button>
              </div>
            </div>
          </div>
        </div>

        <aside class="space-y-6">
          <div class="card p-5">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">提醒状态</h2>
            <dl class="mt-4 space-y-3 text-sm">
              <div class="flex items-center justify-between gap-3">
                <dt class="text-gray-500 dark:text-dark-400">当前状态</dt>
                <dd><span class="badge" :class="statusBadgeClass(status?.state)">{{ statusLabel(status?.state) }}</span></dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-gray-500 dark:text-dark-400">剩余天数</dt>
                <dd class="font-medium text-gray-900 dark:text-white">{{ daysLabel }}</dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-gray-500 dark:text-dark-400">下次提醒</dt>
                <dd class="font-medium text-gray-900 dark:text-white">{{ status?.next_reminder || '-' }}</dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-gray-500 dark:text-dark-400">上次通知</dt>
                <dd class="font-medium text-gray-900 dark:text-white">{{ status?.last_notified_at || '-' }}</dd>
              </div>
            </dl>
          </div>

          <div class="rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300">
            本页只保存续费提醒所需的非敏感信息，不保存服务器登录凭据。
          </div>
        </aside>
      </section>

      <section v-else-if="activeTab === 'server'" class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">服务器信息</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">用于确认是哪台服务器需要续费，不记录任何登录凭据。</p>
        </div>
        <div class="grid gap-6 p-5 xl:grid-cols-[minmax(0,0.9fr)_minmax(420px,1.1fr)]">
          <div class="rounded-md border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0">
                <h3 class="truncate text-lg font-semibold text-gray-900 dark:text-white">{{ form.server_name || '未命名服务器' }}</h3>
                <div class="mt-2 flex flex-wrap gap-2">
                  <span class="badge" :class="form.enabled ? 'badge-success' : 'badge-gray'">{{ form.enabled ? '提醒已开启' : '提醒已关闭' }}</span>
                  <span class="badge" :class="statusBadgeClass(status?.state)">{{ statusLabel(status?.state) }}</span>
                </div>
              </div>
              <button type="button" class="btn btn-secondary btn-sm shrink-0" :disabled="!form.panel_url" @click="openRenewalPanel">
                <Icon name="externalLink" size="xs" />
                面板
              </button>
            </div>
            <dl class="mt-5 grid gap-3 text-sm">
              <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-900/50">
                <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">服务商</dt>
                <dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ form.provider || '-' }}</dd>
              </div>
              <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-900/50">
                <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">Host ID</dt>
                <dd class="mt-1 font-mono font-medium text-gray-900 dark:text-white">{{ form.host_id || '-' }}</dd>
              </div>
              <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-900/50">
                <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">IP</dt>
                <dd class="mt-1 font-mono font-medium text-gray-900 dark:text-white">{{ form.ip_address || '-' }}</dd>
              </div>
              <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-900/50">
                <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">系统</dt>
                <dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ form.operating_system || '-' }}</dd>
              </div>
              <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-900/50">
                <dt class="text-xs font-medium text-gray-500 dark:text-dark-400">到期时间</dt>
                <dd class="mt-1 font-medium text-gray-900 dark:text-white">{{ expiryDisplay }}</dd>
              </div>
            </dl>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <label class="block">
              <span class="input-label">服务器名称</span>
              <input v-model.trim="form.server_name" class="input" placeholder="例如：美国1区精品网 4H4G" />
            </label>
            <label class="block">
              <span class="input-label">服务商</span>
              <input v-model.trim="form.provider" class="input" placeholder="例如：七云 / Client" />
            </label>
            <label class="block">
              <span class="input-label">Host ID</span>
              <input v-model.trim="form.host_id" class="input" placeholder="服务器面板里的 Host ID" />
            </label>
            <label class="block">
              <span class="input-label">IP 地址</span>
              <input v-model.trim="form.ip_address" class="input font-mono" placeholder="服务器公网 IP" autocomplete="off" />
            </label>
            <label class="block md:col-span-2">
              <span class="input-label">系统镜像</span>
              <input v-model.trim="form.operating_system" class="input" placeholder="例如：Ubuntu-22.04-x64" />
            </label>
            <label class="block md:col-span-2">
              <span class="input-label">续费面板地址</span>
              <input v-model.trim="form.panel_url" class="input" placeholder="https://idc.example.com" autocomplete="off" />
            </label>
            <label class="block">
              <span class="input-label">到期日期</span>
              <input v-model="form.expires_at" type="date" class="input" />
            </label>
            <label class="block">
              <span class="input-label">到期时间</span>
              <input v-model="form.expires_at_time" type="time" step="1" class="input" />
            </label>
          </div>
        </div>
      </section>

      <section v-else class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">提醒配置</h2>
          </div>
          <div class="space-y-4 p-5">
            <div class="flex items-center justify-between gap-4 rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700">
              <div>
                <span class="block text-sm font-medium text-gray-900 dark:text-white">启用提醒</span>
                <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">开启后按提前提醒天数检查到期状态。</span>
              </div>
              <div class="flex items-center gap-3">
                <span class="text-sm font-medium" :class="form.enabled ? 'text-primary-700 dark:text-primary-300' : 'text-gray-500 dark:text-dark-400'">
                  {{ form.enabled ? '已开启' : '已关闭' }}
                </span>
                <Toggle v-model="form.enabled" class="scale-110" />
              </div>
            </div>

            <label class="block">
              <span class="input-label">提前提醒天数</span>
              <input v-model.trim="reminderDaysText" class="input" placeholder="7,3,1" />
            </label>
          </div>
        </div>

        <aside class="card p-5">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">规则预览</h2>
          <dl class="mt-4 space-y-3 text-sm">
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">提醒对象</dt>
              <dd class="font-medium text-gray-900 dark:text-white">{{ form.server_name || '-' }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">到期时间</dt>
              <dd class="font-medium text-gray-900 dark:text-white">{{ expiryDisplay }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">提前天数</dt>
              <dd class="font-medium text-gray-900 dark:text-white">{{ reminderDaysText || '-' }}</dd>
            </div>
          </dl>
        </aside>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores/app'
import { adminPlusAPI, type ServerRenewalStatus } from '@/api/admin/adminPlus'

type TabValue = 'dashboard' | 'server' | 'reminders'

const appStore = useAppStore()

const tabs: Array<{ value: TabValue; label: string }> = [
  { value: 'dashboard', label: '工作台' },
  { value: 'server', label: '服务器信息' },
  { value: 'reminders', label: '提醒配置' }
]

const activeTab = ref<TabValue>('dashboard')
const loading = ref(false)
const saving = ref(false)
const status = ref<ServerRenewalStatus | null>(null)
const reminderDaysText = ref('7,3,1')

const form = reactive<ServerRenewalStatus>({
  enabled: true,
  server_name: 'sub2api-admin-plus',
  provider: '',
  host_id: '',
  ip_address: '',
  operating_system: '',
  panel_url: '',
  expires_at: '',
  expires_at_time: '',
  reminder_days: [7, 3, 1],
  days_remaining: 0,
  state: 'unconfigured'
})

const daysLabel = computed(() => {
  const value = status.value
  if (!value?.expires_at) return '未配置'
  if (value.days_remaining < 0) return `逾期 ${Math.abs(value.days_remaining)} 天`
  if (value.days_remaining === 0) return '今天'
  return `${value.days_remaining} 天`
})

const expiryDisplay = computed(() => {
  if (!form.expires_at) return '未配置'
  return [form.expires_at, form.expires_at_time].filter(Boolean).join(' ')
})

const stateTextClass = computed(() => statusTextClass(status.value?.state))

onMounted(() => {
  void loadPage()
})

async function loadPage() {
  loading.value = true
  try {
    const nextStatus = await adminPlusAPI.getServerRenewal()
    applyStatus(nextStatus)
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || '加载续费提醒失败')
  } finally {
    loading.value = false
  }
}

async function saveRenewal() {
  saving.value = true
  try {
    const nextStatus = await adminPlusAPI.updateServerRenewal({
      enabled: form.enabled,
      server_name: form.server_name.trim() || 'sub2api-admin-plus',
      provider: form.provider?.trim() || '',
      host_id: form.host_id?.trim() || '',
      ip_address: form.ip_address?.trim() || '',
      operating_system: form.operating_system?.trim() || '',
      panel_url: form.panel_url?.trim() || '',
      expires_at: form.expires_at.trim(),
      expires_at_time: form.expires_at_time?.trim() || '',
      reminder_days: parseReminderDays(reminderDaysText.value)
    })
    applyStatus(nextStatus)
    appStore.showSuccess('续费提醒已保存')
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || '保存续费提醒失败')
  } finally {
    saving.value = false
  }
}

function applyStatus(nextStatus: ServerRenewalStatus) {
  status.value = nextStatus
  Object.assign(form, {
    enabled: Boolean(nextStatus.enabled),
    server_name: nextStatus.server_name || 'sub2api-admin-plus',
    provider: nextStatus.provider || '',
    host_id: nextStatus.host_id || '',
    ip_address: nextStatus.ip_address || '',
    operating_system: nextStatus.operating_system || '',
    panel_url: nextStatus.panel_url || '',
    expires_at: nextStatus.expires_at || '',
    expires_at_time: nextStatus.expires_at_time || '',
    reminder_days: nextStatus.reminder_days?.length ? nextStatus.reminder_days : [7, 3, 1],
    days_remaining: nextStatus.days_remaining || 0,
    state: nextStatus.state || 'unconfigured',
    next_reminder: nextStatus.next_reminder || '',
    last_notified_at: nextStatus.last_notified_at || ''
  })
  reminderDaysText.value = form.reminder_days.join(',')
}

function parseReminderDays(value: string): number[] {
  const days = value
    .split(/[,\s]+/)
    .map((item) => Number.parseInt(item, 10))
    .filter((item) => Number.isFinite(item) && item >= 0 && item <= 365)
  const unique = Array.from(new Set(days)).sort((a, b) => b - a)
  return unique.length ? unique : [7, 3, 1]
}

function openRenewalPanel() {
  const url = normalizePanelURL(form.panel_url)
  if (!url) {
    appStore.showError('请先填写续费面板地址')
    activeTab.value = 'server'
    return
  }
  window.open(url, '_blank', 'noopener,noreferrer')
}

function normalizePanelURL(value?: string): string {
  const trimmed = value?.trim() || ''
  if (!trimmed) return ''
  if (/^https?:\/\//i.test(trimmed)) return trimmed
  return `https://${trimmed}`
}

function statusLabel(value?: string): string {
  if (value === 'active') return '正常'
  if (value === 'reminder_due') return '待提醒'
  if (value === 'due_today') return '今日到期'
  if (value === 'expired') return '已到期'
  return '未配置'
}

function statusBadgeClass(value?: string): string {
  if (value === 'active') return 'badge-success'
  if (value === 'reminder_due' || value === 'due_today') return 'badge-warning'
  if (value === 'expired') return 'badge-danger'
  return 'badge-gray'
}

function statusTextClass(value?: string): string {
  if (value === 'active') return 'text-emerald-700 dark:text-emerald-400'
  if (value === 'reminder_due' || value === 'due_today') return 'text-amber-600 dark:text-amber-400'
  if (value === 'expired') return 'text-rose-600 dark:text-rose-400'
  return 'text-gray-700 dark:text-dark-200'
}
</script>
