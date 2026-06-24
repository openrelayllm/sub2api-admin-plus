<template>
  <div class="flex flex-wrap-reverse items-start justify-between gap-3">
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative w-full sm:w-64">
        <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
        <input :value="filters.q" class="input pl-9" placeholder="搜索账号..." @input="updateFilter('q', ($event.target as HTMLInputElement).value)" />
      </div>
      <select :value="filters.platform" class="input w-40" @change="updateFilter('platform', ($event.target as HTMLSelectElement).value)">
        <option value="">全部平台</option>
        <option value="openai">OpenAI</option>
        <option value="anthropic">Claude</option>
        <option value="gemini">Gemini</option>
        <option value="antigravity">Antigravity</option>
      </select>
      <select :value="filters.type" class="input w-40" @change="updateFilter('type', ($event.target as HTMLSelectElement).value)">
        <option value="">全部类型</option>
        <option v-for="option in typeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
	      <select :value="filters.status" class="input w-40" @change="updateFilter('status', ($event.target as HTMLSelectElement).value)">
	        <option value="">全部状态</option>
	        <option value="active">正常</option>
	        <option value="disabled">停用</option>
	        <option value="error">异常</option>
	        <option value="rate_limited">限流</option>
	        <option value="temp_unschedulable">临时不可调度</option>
	      </select>
      <select :value="filters.group" class="input w-40" @change="updateFilter('group', ($event.target as HTMLSelectElement).value)">
        <option value="">全部分组</option>
        <option value="ungrouped">未分组</option>
        <option v-for="option in groupOptions" :key="option" :value="option">{{ option }}</option>
      </select>
      <select :value="selectedSupplierId" class="input w-44" @change="$emit('update:selectedSupplierId', Number(($event.target as HTMLSelectElement).value))">
        <option :value="0">全部供应商</option>
        <option v-for="supplier in suppliers" :key="supplier.id" :value="supplier.id">{{ supplier.name }}</option>
      </select>
    </div>

    <div class="flex flex-wrap items-center justify-end gap-2">
      <button type="button" class="btn btn-secondary px-2 md:px-3" :disabled="loading" title="刷新" @click="$emit('refresh')">
        <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
      </button>
      <div class="relative">
        <button type="button" class="btn btn-secondary px-2 md:px-3" :title="autoRefreshEnabled ? `自动刷新 ${autoRefreshCountdown}s` : '自动刷新'" @click="toggleAutoRefreshMenu">
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': autoRefreshEnabled }" />
          <span class="hidden md:inline">{{ autoRefreshEnabled ? `${autoRefreshCountdown}s` : '自动刷新' }}</span>
        </button>
        <div v-if="autoRefreshOpen" class="absolute right-0 z-50 mt-2 w-56 rounded-lg border border-gray-200 bg-white p-2 shadow-lg dark:border-dark-700 dark:bg-dark-800">
          <button class="menu-item justify-between" @click="setAutoRefreshEnabled(!autoRefreshEnabled)">
            <span>启用自动刷新</span>
            <Icon v-if="autoRefreshEnabled" name="check" size="sm" class="text-primary-500" />
          </button>
          <div class="my-1 border-t border-gray-100 dark:border-dark-700" />
          <button v-for="seconds in autoRefreshIntervals" :key="seconds" class="menu-item justify-between" @click="setAutoRefreshInterval(seconds)">
            <span>{{ seconds }} 秒</span>
            <Icon v-if="autoRefreshInterval === seconds" name="check" size="sm" class="text-primary-500" />
          </button>
        </div>
      </div>
      <div class="relative">
        <button type="button" class="btn btn-secondary px-2 md:px-3" @click="toolsOpen = !toolsOpen">
          <Icon name="more" size="sm" />
          <span class="hidden md:inline">更多操作</span>
          <Icon name="chevronDown" size="xs" class="hidden md:inline" />
        </button>
        <div v-if="toolsOpen" class="absolute right-0 z-50 mt-2 w-72 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-xl dark:border-dark-700 dark:bg-dark-800">
          <div class="p-2">
            <div class="px-2 py-2 text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-dark-400">筛选与视图</div>
            <button class="menu-item" @click="emitAndClose('resetFilters')">
              <span class="menu-icon bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300"><Icon name="x" size="sm" /></span>
              <span>清除筛选</span>
            </button>
            <button class="menu-item" @click="emitAndClose('selectCurrentPage')">
              <span class="menu-icon bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300"><Icon name="check" size="sm" /></span>
              <span>选择当前页</span>
            </button>
            <button class="menu-item" @click="emitAndClose('clearSelection')">
              <span class="menu-icon bg-amber-50 text-amber-600 dark:bg-amber-900/30 dark:text-amber-300"><Icon name="ban" size="sm" /></span>
              <span>清空选择</span>
            </button>
            <button class="menu-item" @click="emitAndClose('createAccount')">
              <span class="menu-icon bg-emerald-50 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-300"><Icon name="externalLink" size="sm" /></span>
              <span>前往开通账号</span>
            </button>
            <div class="my-2 border-t border-gray-100 dark:border-dark-700" />
            <div class="px-2 py-2">
              <div class="flex items-center justify-between gap-3">
                <span class="text-xs font-semibold uppercase tracking-wide text-gray-400 dark:text-dark-400">显示列</span>
                <Icon name="grid" size="sm" class="text-gray-400" />
              </div>
            </div>
            <button v-for="column in columnOptions" :key="column.key" class="menu-item justify-between" @click="$emit('toggleColumn', column.key)">
              <span class="truncate">{{ column.label }}</span>
              <Icon v-if="visibleColumnKeys.includes(column.key)" name="check" size="sm" class="text-primary-500" />
            </button>
          </div>
        </div>
      </div>
      <button type="button" class="btn btn-primary" @click="$emit('createAccount')">添加账号</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { Supplier } from '@/api/admin/adminPlus'

interface ToolbarFilters {
  q: string
  platform: string
  type: string
  status: string
  group: string
}

interface ColumnOption {
  key: string
  label: string
}

const props = defineProps<{
  filters: ToolbarFilters
  selectedSupplierId: number
  suppliers: Supplier[]
  typeOptions: Array<{ value: string; label: string }>
  groupOptions: string[]
  columnOptions: ColumnOption[]
  visibleColumnKeys: string[]
  loading: boolean
}>()

const emit = defineEmits<{
  'update:filters': [filters: ToolbarFilters]
  'update:selectedSupplierId': [value: number]
  refresh: []
	  resetFilters: []
	  selectCurrentPage: []
	  clearSelection: []
	  createAccount: []
	  toggleColumn: [key: string]
	}>()

const toolsOpen = ref(false)
const autoRefreshOpen = ref(false)
const autoRefreshEnabled = ref(false)
const autoRefreshInterval = ref(60)
const autoRefreshCountdown = ref(60)
const autoRefreshIntervals = [30, 60, 120]
let autoRefreshTimer: ReturnType<typeof setInterval> | null = null

function updateFilter(key: keyof ToolbarFilters, value: string) {
  emit('update:filters', { ...props.filters, [key]: value })
}

function emitAndClose(event: 'resetFilters' | 'selectCurrentPage' | 'clearSelection' | 'createAccount') {
  toolsOpen.value = false
  if (event === 'resetFilters') emit('resetFilters')
  if (event === 'selectCurrentPage') emit('selectCurrentPage')
  if (event === 'clearSelection') emit('clearSelection')
  if (event === 'createAccount') emit('createAccount')
}

function toggleAutoRefreshMenu() {
  autoRefreshOpen.value = !autoRefreshOpen.value
  toolsOpen.value = false
}

function setAutoRefreshEnabled(enabled: boolean) {
  autoRefreshEnabled.value = enabled
  autoRefreshOpen.value = false
  autoRefreshCountdown.value = autoRefreshInterval.value
  syncAutoRefreshTimer()
}

function setAutoRefreshInterval(seconds: number) {
  autoRefreshInterval.value = seconds
  autoRefreshCountdown.value = seconds
  autoRefreshOpen.value = false
  syncAutoRefreshTimer()
}

function syncAutoRefreshTimer() {
  if (autoRefreshTimer) {
    clearInterval(autoRefreshTimer)
    autoRefreshTimer = null
  }
  if (!autoRefreshEnabled.value) return
  autoRefreshTimer = setInterval(() => {
    autoRefreshCountdown.value -= 1
    if (autoRefreshCountdown.value > 0) return
    autoRefreshCountdown.value = autoRefreshInterval.value
    emit('refresh')
  }, 1000)
}

onUnmounted(() => {
  if (autoRefreshTimer) clearInterval(autoRefreshTimer)
})
</script>

<style scoped>
.menu-item {
  @apply flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm text-gray-700 transition-colors hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700;
}

.menu-icon {
  @apply flex h-7 w-7 shrink-0 items-center justify-center rounded-md;
}
</style>
