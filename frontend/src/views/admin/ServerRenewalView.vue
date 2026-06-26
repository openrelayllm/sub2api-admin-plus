<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">续费提醒</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            提前提醒月付服务器续费，避免服务器到期导致服务不可用。
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
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">到期日</p>
          <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ form.expires_at || '未配置' }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ form.provider || '未知服务商' }}</p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-dark-400">下次提醒</p>
          <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ status?.next_reminder || '-' }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ reminderDaysText || '7,3,1' }} 天</p>
        </div>
      </section>

      <section class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
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
              <span class="input-label">服务器名称</span>
              <input v-model.trim="form.server_name" class="input" placeholder="sub2api-admin-plus" />
            </label>
            <label class="block">
              <span class="input-label">服务商</span>
              <input v-model.trim="form.provider" class="input" placeholder="VPS / Cloud Provider" />
            </label>
            <label class="block">
              <span class="input-label">到期日</span>
              <input v-model="form.expires_at" type="date" class="input" />
            </label>
            <label class="block">
              <span class="input-label">提前提醒天数</span>
              <input v-model.trim="reminderDaysText" class="input" placeholder="7,3,1" />
            </label>
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
            </dl>
          </div>

          <div class="rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300">
            续费提醒只负责服务器到期提醒，不参与数据库备份、恢复或对象存储配置。
          </div>
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

const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const status = ref<ServerRenewalStatus | null>(null)
const reminderDaysText = ref('7,3,1')

const form = reactive<ServerRenewalStatus>({
  enabled: true,
  server_name: 'sub2api-admin-plus',
  provider: '',
  expires_at: '',
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
      provider: form.provider.trim(),
      expires_at: form.expires_at.trim(),
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
    expires_at: nextStatus.expires_at || '',
    reminder_days: nextStatus.reminder_days?.length ? nextStatus.reminder_days : [7, 3, 1],
    days_remaining: nextStatus.days_remaining || 0,
    state: nextStatus.state || 'unconfigured',
    next_reminder: nextStatus.next_reminder || ''
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
