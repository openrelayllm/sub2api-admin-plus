<template>
  <section class="grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_360px]">
    <div class="card overflow-hidden">
      <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">计划配置</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">配置调度频率、采集窗口、执行策略并查看最近运行结果。</p>
          </div>
          <div class="flex flex-wrap gap-2 text-xs">
            <span class="badge badge-success">{{ healthyCount }} 正常</span>
            <span class="badge" :class="issueCount > 0 ? 'badge-danger' : 'badge-gray'">{{ issueCount }} 问题</span>
            <span class="badge badge-gray">{{ enabledCount }} 启用</span>
          </div>
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[1180px] divide-y divide-gray-200 dark:divide-dark-700">
          <thead class="bg-gray-50 dark:bg-dark-800">
            <tr>
              <th class="w-[250px] px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">计划</th>
              <th class="w-[110px] px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">状态</th>
              <th class="w-[150px] px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">频率/窗口</th>
              <th class="w-[150px] px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">范围</th>
              <th class="w-[150px] px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">策略</th>
              <th class="w-[210px] px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">运行</th>
              <th class="w-[180px] px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">问题概览</th>
              <th class="w-[210px] px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">操作</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
            <tr v-if="plans.length === 0">
              <td colspan="8" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无计划配置。</td>
            </tr>
            <tr v-for="plan in plans" :key="plan.id">
              <td class="px-4 py-4 align-top">
                <div class="flex items-center gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ plan.name }}</span>
                  <span v-if="plan.high_cost" class="badge badge-warning">高成本</span>
                </div>
                <p class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-dark-400">{{ plan.description }}</p>
              </td>
              <td class="px-4 py-4 align-top">
                <span class="badge" :class="planStatusClass(plan.status)">{{ planStatusLabel(plan.status) }}</span>
              </td>
              <td class="px-4 py-4 align-top text-sm text-gray-500 dark:text-dark-400">
                <p class="font-medium text-gray-700 dark:text-gray-200">{{ plan.frequency_label }}</p>
                <p class="mt-1 text-xs">{{ plan.window_minutes }} 分钟窗口</p>
              </td>
              <td class="px-4 py-4 align-top text-sm text-gray-500 dark:text-dark-400">{{ plan.scope }}</td>
              <td class="px-4 py-4 align-top text-sm text-gray-500 dark:text-dark-400">
                <p>{{ misfireLabel(plan.misfire_policy) }}</p>
                <p class="mt-1 text-xs">{{ concurrencyLabel(plan.concurrency_policy) }}</p>
              </td>
              <td class="px-4 py-4 align-top text-sm text-gray-500 dark:text-dark-400">
                <p>
                  <span class="text-xs text-gray-400 dark:text-dark-500">上次成功</span>
                  <span class="ml-2 font-medium text-gray-800 dark:text-gray-100">{{ formatDateTime(plan.last_success_at) || '-' }}</span>
                </p>
                <p class="mt-1">
                  <span class="text-xs text-gray-400 dark:text-dark-500">下次运行</span>
                  <span class="ml-2">{{ planNextRunLabel(plan) }}</span>
                </p>
              </td>
              <td class="px-4 py-4 align-top">
                <div v-if="plan.issue_count > 0" class="max-w-[170px]">
                  <span class="badge badge-danger">{{ plan.issue_count }} 个问题</span>
                  <p class="mt-2 truncate text-xs text-gray-500 dark:text-dark-400" :title="plan.last_issue || ''">{{ plan.last_issue || '待查看运行详情' }}</p>
                  <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">{{ formatDateTime(plan.last_issue_at) || '-' }}</p>
                </div>
                <span v-else class="badge badge-success">无异常</span>
              </td>
              <td class="px-4 py-4 align-top">
                <div class="flex flex-wrap gap-2">
                  <button type="button" class="btn btn-secondary btn-sm" :disabled="running || planManualTaskTypes(plan).length === 0" @click="emit('run', plan)">
                    <Icon name="play" size="xs" />
                    运行
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm" :disabled="updatingPlanId === plan.id" @click="openEdit(plan)">
                    设置
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm" :disabled="updatingPlanId === plan.id" @click="emit('status', plan, plan.status === 'enabled' ? 'paused' : 'enabled')">
                    {{ plan.status === 'enabled' ? '暂停' : '启用' }}
                  </button>
                  <button v-if="plan.status !== 'disabled'" type="button" class="btn btn-secondary btn-sm" :disabled="updatingPlanId === plan.id" @click="emit('status', plan, 'disabled')">
                    停用
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <aside class="card p-5">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">计划向导</h2>
      <ol class="mt-4 space-y-4">
        <li v-for="(step, index) in schedulerWizardSteps" :key="step" class="flex gap-3">
          <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary-100 text-sm font-semibold text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">{{ index + 1 }}</span>
          <span class="text-sm text-gray-700 dark:text-gray-200">{{ step }}</span>
        </li>
      </ol>
    </aside>
  </section>

  <BaseDialog :show="Boolean(editingPlan)" :title="editingPlan ? `设置计划 - ${editingPlan.name}` : '设置计划'" width="wide" @close="editingPlan = null">
    <form class="grid gap-4 md:grid-cols-2" @submit.prevent="saveEdit">
      <label class="block">
        <span class="input-label">状态</span>
        <select v-model="form.status" class="input">
          <option value="enabled">启用</option>
          <option value="paused">暂停</option>
          <option value="disabled">停用</option>
        </select>
      </label>
      <label class="block">
        <span class="input-label">频率</span>
        <div class="flex items-center gap-2">
          <input v-model.number="intervalMinutes" type="number" min="0" max="43200" step="1" class="input" />
          <span class="shrink-0 text-sm text-gray-500 dark:text-dark-400">分钟</span>
        </div>
      </label>
      <label class="block">
        <span class="input-label">采集窗口</span>
        <div class="flex items-center gap-2">
          <input v-model.number="form.window_minutes" type="number" min="1" max="1440" step="1" class="input" />
          <span class="shrink-0 text-sm text-gray-500 dark:text-dark-400">分钟</span>
        </div>
      </label>
      <label class="block">
        <span class="input-label">范围</span>
        <input v-model.trim="form.scope" type="text" class="input" />
      </label>
      <label class="block">
        <span class="input-label">Misfire</span>
        <select v-model="form.misfire_policy" class="input">
          <option value="fire_once">只补一次</option>
          <option value="backfill">回填窗口</option>
          <option value="skip">跳过</option>
        </select>
      </label>
      <label class="block">
        <span class="input-label">并发</span>
        <select v-model="form.concurrency_policy" class="input">
          <option value="forbid">禁止重叠</option>
          <option value="allow">允许重叠</option>
        </select>
      </label>
      <div class="flex justify-end gap-2 md:col-span-2">
        <button type="button" class="btn btn-secondary" @click="editingPlan = null">取消</button>
        <button type="submit" class="btn btn-primary" :disabled="Boolean(editingPlan && updatingPlanId === editingPlan.id)">保存</button>
      </div>
    </form>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { SchedulerPlan, SchedulerPlanConfig } from '@/api/admin/adminPlus'
import { formatDateTime, planManualTaskTypes, planStatusClass, planStatusLabel, schedulerWizardSteps } from './presentation'

const props = defineProps<{
  plans: SchedulerPlan[]
  running: boolean
  updatingPlanId: string | null
}>()

const emit = defineEmits<{
  (event: 'run', plan: SchedulerPlan): void
  (event: 'status', plan: SchedulerPlan, status: 'enabled' | 'paused' | 'disabled'): void
  (event: 'save', plan: SchedulerPlan, config: SchedulerPlanConfig): void
}>()

const editingPlan = ref<SchedulerPlan | null>(null)
const intervalMinutes = ref(0)
const form = reactive<SchedulerPlanConfig>({
  status: 'enabled',
  scope: '',
  interval_seconds: 0,
  window_minutes: 10,
  misfire_policy: 'fire_once',
  concurrency_policy: 'forbid'
})

const enabledCount = computed(() => props.plans.filter((plan) => plan.status === 'enabled').length)
const issueCount = computed(() => props.plans.reduce((sum, plan) => sum + (plan.issue_count || 0), 0))
const healthyCount = computed(() => props.plans.filter((plan) => (plan.issue_count || 0) === 0).length)

function openEdit(plan: SchedulerPlan) {
  editingPlan.value = plan
  intervalMinutes.value = Math.round((plan.interval_seconds || 0) / 60)
  form.status = normalizeStatus(plan.status)
  form.scope = plan.scope || ''
  form.interval_seconds = plan.interval_seconds || 0
  form.window_minutes = plan.window_minutes || 10
  form.misfire_policy = plan.misfire_policy || 'fire_once'
  form.concurrency_policy = plan.concurrency_policy || 'forbid'
}

function saveEdit() {
  if (!editingPlan.value) return
  emit('save', editingPlan.value, {
    ...form,
    interval_seconds: Math.max(0, Number(intervalMinutes.value || 0)) * 60,
    window_minutes: Math.max(1, Number(form.window_minutes || 1))
  })
  editingPlan.value = null
}

function normalizeStatus(status: string): 'enabled' | 'paused' | 'disabled' {
  if (status === 'paused' || status === 'disabled') return status
  return 'enabled'
}

function misfireLabel(value: string): string {
  return {
    fire_once: '只补一次',
    backfill: '回填窗口',
    skip: '跳过'
  }[value] || value
}

function concurrencyLabel(value: string): string {
  return {
    forbid: '禁止重叠',
    allow: '允许重叠'
  }[value] || value
}

function planNextRunLabel(plan: SchedulerPlan): string {
  if (!plan.next_run_at) return '-'
  const formatted = formatDateTime(plan.next_run_at) || '-'
  if (new Date(plan.next_run_at).getTime() <= Date.now()) {
    return `待调度 · ${formatted}`
  }
  return formatted
}
</script>
