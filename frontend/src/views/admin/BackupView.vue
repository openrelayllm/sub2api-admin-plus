<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">数据备份</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            PostgreSQL 数据库备份、对象存储、续费提醒和运维历史清理。
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button type="button" class="btn btn-secondary" :disabled="refreshing" @click="refreshPage">
            <Icon name="refresh" size="sm" />
            刷新
          </button>
          <button type="button" class="btn btn-secondary" :disabled="creatingBackup || !storageConfigured" @click="createManualBackup">
            <Icon name="database" size="sm" />
            {{ creatingBackup ? '提交中...' : '重新备份' }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="saving" @click="saveSettings">
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

      <section v-if="activeTab === 'overview'" class="space-y-6">
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">对象存储</p>
            <p class="mt-2 text-2xl font-semibold" :class="storageConfigured ? 'text-emerald-700 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">
              {{ storageConfigured ? '已配置' : '待配置' }}
            </p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ providerLabel(s3Form.provider) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">定时备份</p>
            <p class="mt-2 text-2xl font-semibold" :class="scheduleForm.enabled ? 'text-emerald-700 dark:text-emerald-400' : 'text-gray-700 dark:text-dark-200'">
              {{ scheduleForm.enabled ? '启用' : '停用' }}
            </p>
            <p class="mt-1 font-mono text-xs text-gray-500 dark:text-dark-400">{{ scheduleForm.cron_expr || '-' }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">最近成功</p>
            <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ latestSuccessTime }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ latestSuccessSize }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">服务器续费</p>
            <p class="mt-2 text-2xl font-semibold" :class="renewalTextClass">{{ renewalDaysLabel }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ renewalStateLabel(status?.renewal.state) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">历史清理</p>
            <p class="mt-2 text-2xl font-semibold" :class="cleanupForm.enabled ? 'text-emerald-700 dark:text-emerald-400' : 'text-gray-700 dark:text-dark-200'">
              {{ cleanupForm.enabled ? `${cleanupForm.retain_days} 天` : '停用' }}
            </p>
            <p class="mt-1 font-mono text-xs text-gray-500 dark:text-dark-400">{{ cleanupForm.cron_expr || '-' }}</p>
          </div>
        </div>

        <section class="grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_minmax(320px,0.8fr)]">
          <div class="card overflow-hidden">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">备份工作台</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">只备份 PostgreSQL 中的业务事实数据，不备份缓存、临时文件和无价值运行残留。</p>
            </div>
            <div class="space-y-4 p-5">
              <div
                class="rounded-md border px-4 py-3"
                :class="storageConfigured ? 'border-emerald-200 bg-emerald-50 dark:border-emerald-900/60 dark:bg-emerald-950/30' : 'border-amber-200 bg-amber-50 dark:border-amber-900/60 dark:bg-amber-950/30'"
              >
                <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                  <div>
                    <p class="text-sm font-medium" :class="storageConfigured ? 'text-emerald-800 dark:text-emerald-300' : 'text-amber-800 dark:text-amber-300'">
                      {{ storageConfigured ? '对象存储已就绪，可以重新备份或恢复数据。' : '请先配置对象存储，备份文件会上传到远端。' }}
                    </p>
                    <p class="mt-1 text-xs" :class="storageConfigured ? 'text-emerald-700 dark:text-emerald-400' : 'text-amber-700 dark:text-amber-400'">
                      当前渠道：{{ providerLabel(s3Form.provider) }} · Prefix：{{ s3Form.prefix || 'backups' }}
                    </p>
                  </div>
                  <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="activeTab = 'storage'">
                    {{ storageConfigured ? '查看配置' : '去配置' }}
                  </button>
                </div>
              </div>

              <div class="grid gap-4 md:grid-cols-3">
                <div class="rounded-md border border-gray-200 p-4 dark:border-dark-700">
                  <p class="text-xs font-medium text-gray-500 dark:text-dark-400">运行状态</p>
                  <p class="mt-2 text-lg font-semibold" :class="status?.running ? 'text-amber-600 dark:text-amber-400' : 'text-emerald-700 dark:text-emerald-400'">
                    {{ status?.running ? '备份中' : '空闲' }}
                  </p>
                  <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ status?.running?.progress || '暂无运行任务' }}</p>
                </div>
                <div class="rounded-md border border-gray-200 p-4 dark:border-dark-700">
                  <p class="text-xs font-medium text-gray-500 dark:text-dark-400">最近成功</p>
                  <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ latestSuccessTime }}</p>
                  <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ latestSuccessSize }}</p>
                </div>
                <div class="rounded-md border border-gray-200 p-4 dark:border-dark-700">
                  <p class="text-xs font-medium text-gray-500 dark:text-dark-400">最近失败</p>
                  <p class="mt-2 text-lg font-semibold" :class="status?.latest_failure ? 'text-rose-600 dark:text-rose-400' : 'text-gray-900 dark:text-white'">
                    {{ formatDateTime(status?.latest_failure?.finished_at) || '-' }}
                  </p>
                  <p class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">{{ status?.latest_failure?.error_message || '暂无失败记录' }}</p>
                </div>
              </div>
            </div>
          </div>

          <aside class="space-y-6">
            <div class="card p-5">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">快捷操作</h2>
              <div class="mt-4 grid gap-2">
                <button type="button" class="btn btn-primary w-full justify-center" :disabled="creatingBackup || !storageConfigured" @click="createManualBackup">
                  <Icon name="database" size="sm" />
                  {{ creatingBackup ? '提交中...' : '重新备份当前数据' }}
                </button>
                <button type="button" class="btn btn-secondary w-full justify-center" @click="activeTab = 'records'">
                  查看备份记录
                </button>
                <button type="button" class="btn btn-secondary w-full justify-center" @click="guideOpen = true">
                  配置渠道指引
                </button>
              </div>
            </div>

            <div class="card p-5">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">续费与清理</h2>
              <dl class="mt-4 space-y-3 text-sm">
                <div class="flex items-center justify-between gap-3">
                  <dt class="text-gray-500 dark:text-dark-400">服务器续费</dt>
                  <dd><span class="badge" :class="renewalStateClass(status?.renewal.state)">{{ renewalStateLabel(status?.renewal.state) }}</span></dd>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <dt class="text-gray-500 dark:text-dark-400">到期剩余</dt>
                  <dd class="font-medium text-gray-900 dark:text-white">{{ renewalDaysLabel }}</dd>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <dt class="text-gray-500 dark:text-dark-400">历史清理</dt>
                  <dd class="font-medium text-gray-900 dark:text-white">{{ cleanupForm.enabled ? `${cleanupForm.retain_days} 天以前` : '停用' }}</dd>
                </div>
              </dl>
            </div>
          </aside>
        </section>
      </section>

      <section v-else-if="activeTab === 'storage'" class="card overflow-hidden">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">对象存储</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">默认使用 Cloudflare R2，也兼容标准 S3 和阿里云 OSS 的 S3 接口。</p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="guideOpen = true">
              <Icon name="book" size="sm" />
              配置指引
            </button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="testingStorage" @click="testStorage">
              <Icon name="beaker" size="sm" />
              {{ testingStorage ? '测试中...' : '测试连接' }}
            </button>
          </div>
        </div>
        <div class="grid gap-4 p-5 md:grid-cols-2">
          <label class="block">
            <span class="input-label">存储类型</span>
            <select v-model="s3Form.provider" class="input" @change="applyProviderPreset">
              <option v-for="option in providerOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>
          <label class="block">
            <span class="input-label">Bucket</span>
            <input v-model.trim="s3Form.bucket" class="input" placeholder="sub2api-backups" />
          </label>
          <label class="block md:col-span-2">
            <span class="input-label">Endpoint</span>
            <input v-model.trim="s3Form.endpoint" class="input" :placeholder="storageEndpointPlaceholder" />
          </label>
          <label class="block">
            <span class="input-label">Region</span>
            <input v-model.trim="s3Form.region" class="input" :placeholder="s3Form.provider === 'cloudflare_r2' ? 'auto' : 'us-east-1'" />
          </label>
          <label class="block">
            <span class="input-label">Prefix</span>
            <input v-model.trim="s3Form.prefix" class="input" placeholder="backups" />
          </label>
          <label class="block">
            <span class="input-label">Access Key ID</span>
            <input v-model.trim="s3Form.access_key_id" class="input" autocomplete="off" />
          </label>
          <label class="block">
            <span class="input-label">Secret Access Key</span>
            <input
              v-model.trim="s3Form.secret_access_key"
              class="input"
              type="password"
              autocomplete="new-password"
              :placeholder="s3Form.secret_configured ? '已配置，留空表示不变' : '首次配置必须填写'"
            />
          </label>
          <div class="flex items-center justify-between gap-4 rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700 md:col-span-2">
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">Path Style</span>
              <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">兼容部分 S3 网关和私有对象存储。</span>
            </div>
            <div class="flex items-center gap-3">
              <span class="text-sm font-medium" :class="s3Form.force_path_style ? 'text-primary-700 dark:text-primary-300' : 'text-gray-500 dark:text-dark-400'">
                {{ s3Form.force_path_style ? '已开启' : '已关闭' }}
              </span>
              <Toggle v-model="s3Form.force_path_style" class="scale-110" />
            </div>
          </div>
        </div>
      </section>

      <section v-else-if="activeTab === 'renewal'" class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">服务器续费</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">提前提醒月付服务器续费，避免因到期导致服务不可用。</p>
          </div>
          <div class="space-y-4 p-5">
            <div class="flex items-center justify-between gap-4 rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700">
              <div>
                <span class="block text-sm font-medium text-gray-900 dark:text-white">启用提醒</span>
                <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">开启后按提前提醒天数检查到期状态。</span>
              </div>
              <div class="flex items-center gap-3">
                <span class="text-sm font-medium" :class="renewalForm.enabled ? 'text-primary-700 dark:text-primary-300' : 'text-gray-500 dark:text-dark-400'">
                  {{ renewalForm.enabled ? '已开启' : '已关闭' }}
                </span>
                <Toggle v-model="renewalForm.enabled" class="scale-110" />
              </div>
            </div>
            <label class="block">
              <span class="input-label">服务器名称</span>
              <input v-model.trim="renewalForm.server_name" class="input" placeholder="sub2api-admin-plus" />
            </label>
            <label class="block">
              <span class="input-label">服务商</span>
              <input v-model.trim="renewalForm.provider" class="input" placeholder="Cloudflare / VPS Provider" />
            </label>
            <label class="block">
              <span class="input-label">到期日</span>
              <input v-model="renewalForm.expires_at" type="date" class="input" />
            </label>
            <label class="block">
              <span class="input-label">提前提醒天数</span>
              <input v-model.trim="renewalReminderDaysText" class="input" placeholder="7,3,1" />
            </label>
          </div>
        </div>

        <aside class="card p-5">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">提醒状态</h2>
          <dl class="mt-4 space-y-3 text-sm">
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">当前状态</dt>
              <dd><span class="badge" :class="renewalStateClass(status?.renewal.state)">{{ renewalStateLabel(status?.renewal.state) }}</span></dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">到期剩余</dt>
              <dd class="font-medium text-gray-900 dark:text-white">{{ renewalDaysLabel }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">下次提醒</dt>
              <dd class="font-medium text-gray-900 dark:text-white">{{ status?.renewal.next_reminder || '-' }}</dd>
            </div>
          </dl>
        </aside>
      </section>

      <section v-else-if="activeTab === 'schedule'" class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">定时备份</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">每天自动备份有价值数据，保留周期用于恢复窗口控制。</p>
        </div>
        <div class="space-y-4 p-5">
          <div class="flex items-center justify-between gap-4 rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700">
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">启用定时任务</span>
              <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">建议保持开启，避免服务器故障时缺少可恢复数据。</span>
            </div>
            <div class="flex items-center gap-3">
              <span class="text-sm font-medium" :class="scheduleForm.enabled ? 'text-primary-700 dark:text-primary-300' : 'text-gray-500 dark:text-dark-400'">
                {{ scheduleForm.enabled ? '已开启' : '已关闭' }}
              </span>
              <Toggle v-model="scheduleForm.enabled" class="scale-110" />
            </div>
          </div>
          <label class="block">
            <span class="input-label">Cron</span>
            <input v-model.trim="scheduleForm.cron_expr" class="input font-mono" placeholder="30 3 * * *" />
          </label>
          <div class="grid gap-4 sm:grid-cols-2">
            <label class="block">
              <span class="input-label">保留天数</span>
              <input v-model.number="scheduleForm.retain_days" type="number" min="1" max="365" class="input" />
            </label>
            <label class="block">
              <span class="input-label">保留份数</span>
              <input v-model.number="scheduleForm.retain_count" type="number" min="1" max="500" class="input" />
            </label>
          </div>
        </div>
      </section>

      <section v-else-if="activeTab === 'cleanup'" class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">历史清理</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">定时清理 5 天以前的无价值运维历史，降低重新部署后的恢复噪音。</p>
        </div>
        <div class="space-y-4 p-5">
          <div class="flex items-center justify-between gap-4 rounded-md border border-gray-200 px-4 py-3 dark:border-dark-700">
            <div>
              <span class="block text-sm font-medium text-gray-900 dark:text-white">启用自动清理</span>
              <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">当前清理 5 天以前的运维历史。</span>
            </div>
            <div class="flex items-center gap-3">
              <span class="text-sm font-medium" :class="cleanupForm.enabled ? 'text-primary-700 dark:text-primary-300' : 'text-gray-500 dark:text-dark-400'">
                {{ cleanupForm.enabled ? '已开启' : '已关闭' }}
              </span>
              <Toggle v-model="cleanupForm.enabled" class="scale-110" />
            </div>
          </div>
          <label class="block">
            <span class="input-label">保留天数</span>
            <input v-model.number="cleanupForm.retain_days" type="number" min="1" max="365" class="input" />
          </label>
          <label class="block">
            <span class="input-label">Cron</span>
            <input v-model.trim="cleanupForm.cron_expr" class="input font-mono" placeholder="0 2 * * *" />
          </label>
          <p class="rounded-md bg-gray-50 p-3 text-xs leading-5 text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            清理范围：运维错误日志、告警事件、系统日志、清理审计和指标聚合；不清理用户、账单、用量明细等业务事实。
          </p>
        </div>
      </section>

      <section v-else-if="activeTab === 'records'" class="card overflow-hidden">
        <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">备份记录</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">支持下载、恢复和远端删除备份。</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="creatingBackup || !storageConfigured" @click="createManualBackup">
            <Icon name="database" size="sm" />
            {{ creatingBackup ? '提交中...' : '重新备份' }}
          </button>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[1080px] divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">备份</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">状态</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">大小</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">触发</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">时间</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">恢复</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500 dark:text-dark-400">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="records.length === 0">
                <td colspan="7" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">
                  {{ refreshing ? '加载中...' : '暂无备份记录' }}
                </td>
              </tr>
              <tr v-for="record in records" :key="record.id" class="align-top">
                <td class="px-4 py-4">
                  <div class="font-mono text-xs text-gray-900 dark:text-gray-100">{{ record.id }}</div>
                  <div class="mt-1 max-w-[280px] truncate text-xs text-gray-500 dark:text-dark-400">{{ record.file_name || record.s3_key || '-' }}</div>
                </td>
                <td class="px-4 py-4">
                  <span class="badge" :class="backupStatusClass(record.status)">{{ backupStatusLabel(record.status) }}</span>
                  <div v-if="record.progress" class="mt-2 font-mono text-xs text-gray-500 dark:text-dark-400">{{ record.progress }}</div>
                  <div v-if="record.error_message" class="mt-2 max-w-[260px] whitespace-pre-wrap break-words text-xs text-rose-600 dark:text-rose-400">
                    {{ record.error_message }}
                  </div>
                </td>
                <td class="px-4 py-4 text-sm text-gray-700 dark:text-dark-200">{{ formatBackupBytes(record.size_bytes) }}</td>
                <td class="px-4 py-4 text-sm text-gray-700 dark:text-dark-200">
                  <div>{{ triggerLabel(record.triggered_by) }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ record.backup_type || 'postgres' }}</div>
                </td>
                <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">
                  <div>开始 {{ formatDateTime(record.started_at) || '-' }}</div>
                  <div class="mt-1">完成 {{ formatDateTime(record.finished_at) || '-' }}</div>
                  <div class="mt-1">过期 {{ formatDateTime(record.expires_at) || '-' }}</div>
                </td>
                <td class="px-4 py-4">
                  <span class="badge" :class="restoreStatusClass(record.restore_status)">{{ restoreStatusLabel(record.restore_status) }}</span>
                  <div v-if="record.restore_error" class="mt-2 max-w-[220px] whitespace-pre-wrap break-words text-xs text-rose-600 dark:text-rose-400">
                    {{ record.restore_error }}
                  </div>
                </td>
                <td class="px-4 py-4">
                  <div class="flex justify-end gap-2">
                    <button type="button" class="btn btn-secondary btn-sm" :disabled="record.status !== 'completed'" @click="downloadBackup(record)">
                      <Icon name="download" size="sm" />
                      下载
                    </button>
                    <button type="button" class="btn btn-secondary btn-sm" :disabled="record.status !== 'completed' || record.restore_status === 'running'" @click="openRestoreModal(record)">
                      <Icon name="upload" size="sm" />
                      恢复
                    </button>
                    <button type="button" class="btn btn-danger btn-sm" :disabled="record.status === 'running' || record.restore_status === 'running'" @click="deleteBackupRecord(record)">
                      <Icon name="trash" size="sm" />
                      远端删除
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <transition name="fade">
        <div v-if="guideOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div class="max-h-[90vh] w-full max-w-3xl overflow-hidden rounded-lg bg-white shadow-xl dark:bg-dark-900">
            <div class="flex items-start justify-between gap-4 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ currentStorageGuide.title }}</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ currentStorageGuide.summary }}</p>
              </div>
              <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="guideOpen = false">
                <Icon name="x" size="sm" />
              </button>
            </div>

            <div class="max-h-[calc(90vh-76px)] space-y-5 overflow-y-auto p-5">
              <div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                <a
                  v-for="link in currentStorageGuide.links"
                  :key="link.href"
                  :href="link.href"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="flex items-center justify-between gap-3 rounded-md border border-gray-200 px-3 py-2 text-sm font-medium text-gray-700 hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:text-dark-200 dark:hover:border-primary-500 dark:hover:text-primary-300"
                >
                  <span>{{ link.label }}</span>
                  <Icon name="externalLink" size="sm" />
                </a>
              </div>

              <section>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">配置步骤</h3>
                <ol class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
                  <li v-for="(step, index) in currentStorageGuide.steps" :key="step" class="flex gap-3">
                    <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-700 dark:bg-dark-800 dark:text-dark-200">
                      {{ index + 1 }}
                    </span>
                    <span class="pt-0.5">{{ step }}</span>
                  </li>
                </ol>
              </section>

              <section>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">字段填写</h3>
                <div class="mt-3 overflow-hidden rounded-md border border-gray-200 dark:border-dark-700">
                  <table class="w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
                    <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                      <tr v-for="field in currentStorageGuide.fields" :key="field.name" class="align-top">
                        <td class="w-40 bg-gray-50 px-3 py-3 font-medium text-gray-700 dark:bg-dark-800 dark:text-dark-200">{{ field.name }}</td>
                        <td class="px-3 py-3 text-gray-600 dark:text-dark-300">
                          <div>{{ field.value }}</div>
                          <div v-if="field.example" class="mt-1 font-mono text-xs text-gray-500 dark:text-dark-400">{{ field.example }}</div>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </section>

              <section>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">注意事项</h3>
                <ul class="mt-3 space-y-2 text-sm text-gray-600 dark:text-dark-300">
                  <li v-for="note in currentStorageGuide.notes" :key="note" class="flex gap-2">
                    <span class="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-gray-400 dark:bg-dark-500"></span>
                    <span>{{ note }}</span>
                  </li>
                </ul>
              </section>
            </div>
          </div>
        </div>
      </transition>

      <transition name="fade">
        <div v-if="restoreCandidate" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div class="w-full max-w-lg rounded-lg bg-white shadow-xl dark:bg-dark-900">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">恢复备份</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">恢复会覆盖当前 PostgreSQL 数据库，请确认已经下载或保留可回滚备份。</p>
            </div>
            <div class="space-y-4 p-5">
              <div class="rounded-md bg-gray-50 p-3 font-mono text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">
                {{ restoreCandidate.id }}
              </div>
              <label class="block">
                <span class="input-label">确认文本</span>
                <input v-model.trim="restoreConfirmation" class="input font-mono" placeholder="RESTORE" />
              </label>
            </div>
            <div class="flex justify-end gap-2 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
              <button type="button" class="btn btn-secondary" :disabled="restoring" @click="closeRestoreModal">取消</button>
              <button type="button" class="btn btn-danger" :disabled="restoring || restoreConfirmation !== 'RESTORE'" @click="submitRestore">
                <Icon name="upload" size="sm" />
                {{ restoring ? '提交中...' : '确认恢复' }}
              </button>
            </div>
          </div>
        </div>
      </transition>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores/app'
import { formatBytes, formatDateTime } from '@/utils/format'
import {
  adminPlusAPI,
  type BackupRecord,
  type BackupS3Config,
  type BackupScheduleConfig,
  type BackupSettings,
  type BackupStatus,
  type BackupStorageProvider,
  type HistoryCleanupSettings,
  type ServerRenewalStatus
} from '@/api/admin/adminPlus'

const appStore = useAppStore()

const providerOptions: Array<{ value: BackupStorageProvider; label: string }> = [
  { value: 'cloudflare_r2', label: 'Cloudflare R2' },
  { value: 's3', label: 'S3 兼容' },
  { value: 'aliyun_oss', label: '阿里云 OSS' }
]

type BackupTab = 'overview' | 'storage' | 'renewal' | 'schedule' | 'cleanup' | 'records'

const tabs: Array<{ value: BackupTab; label: string }> = [
  { value: 'overview', label: '工作台' },
  { value: 'storage', label: '对象存储' },
  { value: 'renewal', label: '续费提醒' },
  { value: 'schedule', label: '定时备份' },
  { value: 'cleanup', label: '历史清理' },
  { value: 'records', label: '备份记录' }
]

const activeTab = ref<BackupTab>('overview')
const status = ref<BackupStatus | null>(null)
const records = ref<BackupRecord[]>([])
const refreshing = ref(false)
const saving = ref(false)
const testingStorage = ref(false)
const creatingBackup = ref(false)
const restoring = ref(false)
const guideOpen = ref(false)
const restoreCandidate = ref<BackupRecord | null>(null)
const restoreConfirmation = ref('')
const renewalReminderDaysText = ref('7,3,1')
let pollTimer: ReturnType<typeof window.setInterval> | null = null

interface StorageGuide {
  title: string
  summary: string
  links: Array<{ label: string; href: string }>
  steps: string[]
  fields: Array<{ name: string; value: string; example?: string }>
  notes: string[]
}

const storageGuides: Record<BackupStorageProvider, StorageGuide> = {
  cloudflare_r2: {
    title: 'Cloudflare R2 配置指引',
    summary: '适合默认备份目标。使用 R2 的 S3 API Token，Endpoint 固定包含 Account ID。',
    links: [
      { label: '打开 R2 控制台', href: 'https://dash.cloudflare.com/?to=/:account/r2' },
      { label: '创建 R2 API Token', href: 'https://developers.cloudflare.com/r2/api/tokens/' },
      { label: 'R2 S3 API 文档', href: 'https://developers.cloudflare.com/r2/api/s3/api/' }
    ],
    steps: [
      '进入 Cloudflare Dashboard 的 R2 页面，创建一个用于备份的 bucket。',
      '在 R2 API Tokens 页面创建 Token，权限至少包含目标 bucket 的对象读写。',
      '复制 Access Key ID、Secret Access Key，并在 R2 页面找到 Account ID。',
      '回到本页填写 Endpoint、Bucket、Access Key ID 和 Secret Access Key，然后点击“测试连接”。'
    ],
    fields: [
      { name: 'Endpoint', value: '填写 R2 S3 API endpoint，使用你的 Account ID。', example: 'https://<ACCOUNT_ID>.r2.cloudflarestorage.com' },
      { name: 'Region', value: '固定填写 auto。', example: 'auto' },
      { name: 'Bucket', value: 'R2 bucket 名称。', example: 'sub2api-backups' },
      { name: 'Access Key ID', value: 'R2 API Token 生成的 Access Key ID。' },
      { name: 'Secret Access Key', value: 'R2 API Token 生成的 Secret，只在首次配置或轮换密钥时填写。' },
      { name: 'Path Style', value: '建议开启。' }
    ],
    notes: [
      'R2 bucket 名称和 Prefix 会组成最终对象路径，例如 backups/2026/06/26/*.sql.gz。',
      'Secret 保存后会加密存储，页面不会再次明文展示。',
      '如果测试连接失败，优先检查 Token 权限、Account ID 和 bucket 名称是否一致。'
    ]
  },
  s3: {
    title: 'S3 兼容存储配置指引',
    summary: '适合 AWS S3、MinIO、Backblaze B2、Wasabi 等兼容 S3 API 的对象存储。',
    links: [
      { label: 'AWS S3 控制台', href: 'https://s3.console.aws.amazon.com/s3/home' },
      { label: '创建 S3 Bucket', href: 'https://docs.aws.amazon.com/AmazonS3/latest/userguide/create-bucket-overview.html' },
      { label: 'AWS S3 Endpoint', href: 'https://docs.aws.amazon.com/general/latest/gr/s3.html' },
      { label: 'AWS Access Key', href: 'https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html' }
    ],
    steps: [
      '在对象存储控制台创建 bucket，并记住 bucket 所在 Region。',
      '创建具备目标 bucket 读写权限的 Access Key。',
      '按服务商文档确认 S3 endpoint；AWS 可以使用区域 endpoint，MinIO 等私有存储填写服务地址。',
      '回到本页填写 Endpoint、Region、Bucket 和密钥，然后点击“测试连接”。'
    ],
    fields: [
      { name: 'Endpoint', value: 'AWS 可填区域 endpoint；私有 S3/MinIO 填服务地址。', example: 'https://s3.us-east-1.amazonaws.com' },
      { name: 'Region', value: 'bucket 所在区域。', example: 'us-east-1' },
      { name: 'Bucket', value: 'bucket 名称。', example: 'sub2api-backups' },
      { name: 'Access Key ID', value: 'IAM 或服务商生成的 Access Key ID。' },
      { name: 'Secret Access Key', value: '对应的 Secret Access Key。' },
      { name: 'Path Style', value: 'AWS 通常可关闭；MinIO、私有网关或部分兼容服务建议开启。' }
    ],
    notes: [
      '权限建议最小化到备份 bucket 的 HeadBucket、PutObject、GetObject、DeleteObject。',
      '如果使用 AWS，Region 必须和 bucket 实际所在区域一致。',
      '如果使用私有 S3，确认服务器能从部署环境访问该 Endpoint。'
    ]
  },
  aliyun_oss: {
    title: '阿里云 OSS 配置指引',
    summary: '使用阿里云 OSS 的 S3 兼容接口，Endpoint 使用 s3.oss-{region}.aliyuncs.com 形式。',
    links: [
      { label: '打开 OSS 控制台', href: 'https://oss.console.aliyun.com/overview' },
      { label: '创建 AccessKey', href: 'https://ram.console.aliyun.com/manage/ak' },
      { label: 'OSS Endpoint 文档', href: 'https://help.aliyun.com/zh/oss/user-guide/regions-and-endpoints' },
      { label: 'OSS S3 兼容文档', href: 'https://help.aliyun.com/zh/oss/developer-reference/use-amazon-s3-sdks-to-access-oss' }
    ],
    steps: [
      '进入 OSS 控制台创建 bucket，并确认 bucket 所在地域，例如 cn-hangzhou。',
      '进入 RAM/AccessKey 页面创建用于备份的 AccessKey，建议只授权目标 bucket。',
      '使用 OSS S3 兼容 endpoint，不要填写普通 OSS endpoint。',
      '回到本页填写 Endpoint、Region、Bucket 和密钥，然后点击“测试连接”。'
    ],
    fields: [
      { name: 'Endpoint', value: '填写 OSS S3 兼容 endpoint。', example: 'https://s3.oss-cn-hangzhou.aliyuncs.com' },
      { name: 'Region', value: 'bucket 所在地域。', example: 'cn-hangzhou' },
      { name: 'Bucket', value: 'OSS bucket 名称。', example: 'sub2api-backups' },
      { name: 'Access Key ID', value: '阿里云 AccessKey ID。' },
      { name: 'Secret Access Key', value: '阿里云 AccessKey Secret。' },
      { name: 'Path Style', value: '通常关闭；如果兼容网关要求 path-style，再开启。' }
    ],
    notes: [
      '普通 OSS endpoint 形如 oss-cn-hangzhou.aliyuncs.com；本功能需要 S3 兼容 endpoint，形如 s3.oss-cn-hangzhou.aliyuncs.com。',
      'Region 和 endpoint 的地域必须一致。',
      '建议使用 RAM 子账号和最小权限策略，不要使用主账号 AccessKey。'
    ]
  }
}

const s3Form = reactive<BackupS3Config>({
  provider: 'cloudflare_r2',
  endpoint: '',
  region: 'auto',
  bucket: '',
  access_key_id: '',
  secret_access_key: '',
  secret_configured: false,
  prefix: 'backups',
  force_path_style: true
})

const scheduleForm = reactive<BackupScheduleConfig>({
  enabled: true,
  cron_expr: '30 3 * * *',
  retain_days: 5,
  retain_count: 30
})

const renewalForm = reactive<ServerRenewalStatus>({
  enabled: true,
  server_name: 'sub2api-admin-plus',
  provider: '',
  expires_at: '',
  reminder_days: [7, 3, 1],
  days_remaining: 0,
  state: 'unconfigured'
})

const cleanupForm = reactive<HistoryCleanupSettings>({
  enabled: true,
  retain_days: 5,
  cron_expr: '0 2 * * *'
})

const storageConfigured = computed(() => {
  return Boolean(status.value?.storage_configured || (
    s3Form.bucket &&
    s3Form.access_key_id &&
    (s3Form.secret_configured || s3Form.secret_access_key)
  ))
})

const latestSuccessTime = computed(() => formatDateTime(status.value?.latest_success?.finished_at) || '-')
const latestSuccessSize = computed(() => formatBackupBytes(status.value?.latest_success?.size_bytes || 0))

const renewalDaysLabel = computed(() => {
  const renewal = status.value?.renewal
  if (!renewal?.expires_at) return '未配置'
  if (renewal.days_remaining < 0) return `逾期 ${Math.abs(renewal.days_remaining)} 天`
  if (renewal.days_remaining === 0) return '今天'
  return `${renewal.days_remaining} 天`
})

const renewalTextClass = computed(() => renewalStateTextClass(status.value?.renewal.state))

const storageEndpointPlaceholder = computed(() => {
  if (s3Form.provider === 'cloudflare_r2') return 'https://<account_id>.r2.cloudflarestorage.com'
  if (s3Form.provider === 'aliyun_oss') return 'https://s3.oss-cn-hangzhou.aliyuncs.com'
  return 'https://s3.amazonaws.com'
})

const currentStorageGuide = computed(() => storageGuides[normalizeProvider(s3Form.provider)])

const shouldPoll = computed(() => {
  return Boolean(status.value?.running || records.value.some((item) => item.status === 'running' || item.restore_status === 'running'))
})

onMounted(() => {
  void loadPage()
})

onBeforeUnmount(() => {
  stopPolling()
})

watch(shouldPoll, (enabled) => {
  if (enabled) {
    startPolling()
    return
  }
  stopPolling()
})

async function loadPage(showSpinner = true) {
  if (refreshing.value) return
  refreshing.value = showSpinner
  try {
    const [nextStatus, settings, nextRecords] = await Promise.all([
      adminPlusAPI.getBackupStatus(),
      adminPlusAPI.getBackupSettings(),
      adminPlusAPI.listBackups()
    ])
    status.value = nextStatus
    records.value = nextRecords || []
    applySettings(settings)
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || '加载备份配置失败')
  } finally {
    refreshing.value = false
  }
}

function refreshPage() {
  void loadPage()
}

function applySettings(settings: BackupSettings) {
  Object.assign(s3Form, {
    provider: normalizeProvider(settings.s3.provider),
    endpoint: settings.s3.endpoint || '',
    region: settings.s3.region || (settings.s3.provider === 'aliyun_oss' ? 'cn-hangzhou' : 'auto'),
    bucket: settings.s3.bucket || '',
    access_key_id: settings.s3.access_key_id || '',
    secret_access_key: '',
    secret_configured: Boolean(settings.s3.secret_configured),
    prefix: settings.s3.prefix || 'backups',
    force_path_style: Boolean(settings.s3.force_path_style)
  })
  Object.assign(scheduleForm, {
    enabled: Boolean(settings.schedule.enabled),
    cron_expr: settings.schedule.cron_expr || '30 3 * * *',
    retain_days: normalizeInt(settings.schedule.retain_days, 5),
    retain_count: normalizeInt(settings.schedule.retain_count, 30)
  })
  Object.assign(renewalForm, {
    enabled: settings.renewal.enabled,
    server_name: settings.renewal.server_name || 'sub2api-admin-plus',
    provider: settings.renewal.provider || '',
    expires_at: settings.renewal.expires_at || '',
    reminder_days: settings.renewal.reminder_days?.length ? settings.renewal.reminder_days : [7, 3, 1]
  })
  renewalReminderDaysText.value = (renewalForm.reminder_days || [7, 3, 1]).join(',')
  Object.assign(cleanupForm, {
    enabled: Boolean(settings.cleanup.enabled),
    retain_days: normalizeInt(settings.cleanup.retain_days, 5),
    cron_expr: settings.cleanup.cron_expr || '0 2 * * *',
    description: settings.cleanup.description
  })
}

function applyProviderPreset() {
  if (s3Form.provider === 'cloudflare_r2') {
    if (!s3Form.region || s3Form.region === 'us-east-1' || s3Form.region === 'cn-hangzhou') {
      s3Form.region = 'auto'
    }
    s3Form.force_path_style = true
    return
  }
  if (s3Form.provider === 'aliyun_oss') {
    if (!s3Form.region || s3Form.region === 'auto') {
      s3Form.region = 'cn-hangzhou'
    }
    if (!s3Form.endpoint) {
      s3Form.endpoint = 'https://s3.oss-cn-hangzhou.aliyuncs.com'
    }
    return
  }
  if (!s3Form.region || s3Form.region === 'auto' || s3Form.region === 'cn-hangzhou') {
    s3Form.region = 'us-east-1'
  }
}

async function saveSettings() {
  saving.value = true
  try {
    const settings = await adminPlusAPI.updateBackupSettings({
      s3: buildS3Payload(),
      schedule: {
        enabled: scheduleForm.enabled,
        cron_expr: scheduleForm.cron_expr.trim() || '30 3 * * *',
        retain_days: normalizeInt(scheduleForm.retain_days, 5),
        retain_count: normalizeInt(scheduleForm.retain_count, 30)
      },
      renewal: {
        enabled: Boolean(renewalForm.enabled),
        server_name: String(renewalForm.server_name || '').trim(),
        provider: String(renewalForm.provider || '').trim(),
        expires_at: String(renewalForm.expires_at || '').trim(),
        reminder_days: parseReminderDays(renewalReminderDaysText.value)
      },
      cleanup: {
        enabled: cleanupForm.enabled,
        retain_days: normalizeInt(cleanupForm.retain_days, 5),
        cron_expr: cleanupForm.cron_expr.trim() || '0 2 * * *'
      }
    })
    applySettings(settings)
    status.value = await adminPlusAPI.getBackupStatus()
    appStore.showSuccess('备份配置已保存')
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || '保存备份配置失败')
  } finally {
    saving.value = false
  }
}

async function testStorage() {
  testingStorage.value = true
  try {
    await adminPlusAPI.testBackupStorage(buildS3Payload())
    appStore.showSuccess('对象存储连接正常')
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || '对象存储连接失败')
  } finally {
    testingStorage.value = false
  }
}

async function createManualBackup() {
  creatingBackup.value = true
  try {
    await adminPlusAPI.createBackup({ expire_days: normalizeInt(scheduleForm.retain_days, 5) })
    appStore.showSuccess('重新备份任务已提交')
    await loadPage(false)
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || '提交重新备份任务失败')
  } finally {
    creatingBackup.value = false
  }
}

async function downloadBackup(record: BackupRecord) {
  try {
    const result = await adminPlusAPI.getBackupDownloadURL(record.id)
    window.open(result.url, '_blank', 'noopener')
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || '获取下载链接失败')
  }
}

function openRestoreModal(record: BackupRecord) {
  restoreCandidate.value = record
  restoreConfirmation.value = ''
}

function closeRestoreModal() {
  if (restoring.value) return
  restoreCandidate.value = null
  restoreConfirmation.value = ''
}

async function submitRestore() {
  if (!restoreCandidate.value || restoreConfirmation.value !== 'RESTORE') return
  restoring.value = true
  try {
    await adminPlusAPI.restoreBackup(restoreCandidate.value.id, restoreConfirmation.value)
    appStore.showSuccess('恢复任务已提交')
    restoreCandidate.value = null
    restoreConfirmation.value = ''
    await loadPage(false)
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || '提交恢复任务失败')
  } finally {
    restoring.value = false
  }
}

async function deleteBackupRecord(record: BackupRecord) {
  if (!window.confirm(`确认删除备份 ${record.id}？这会删除对象存储中的远端文件，并移除本地记录。`)) return
  try {
    await adminPlusAPI.deleteBackup(record.id)
    appStore.showSuccess('远端备份已删除')
    await loadPage(false)
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || '删除远端备份失败')
  }
}

function buildS3Payload(): BackupS3Config {
  return {
    provider: normalizeProvider(s3Form.provider),
    endpoint: s3Form.endpoint.trim(),
    region: s3Form.region.trim() || (s3Form.provider === 'cloudflare_r2' ? 'auto' : 'us-east-1'),
    bucket: s3Form.bucket.trim(),
    access_key_id: s3Form.access_key_id.trim(),
    secret_access_key: s3Form.secret_access_key?.trim() || undefined,
    secret_configured: Boolean(s3Form.secret_configured),
    prefix: s3Form.prefix.trim() || 'backups',
    force_path_style: Boolean(s3Form.force_path_style)
  }
}

function startPolling() {
  if (pollTimer) return
  pollTimer = window.setInterval(() => {
    void loadPage(false)
  }, 5000)
}

function stopPolling() {
  if (!pollTimer) return
  window.clearInterval(pollTimer)
  pollTimer = null
}

function parseReminderDays(value: string): number[] {
  const days = value
    .split(/[,\s]+/)
    .map((item) => Number.parseInt(item, 10))
    .filter((item) => Number.isFinite(item) && item >= 0 && item <= 365)
  const unique = Array.from(new Set(days)).sort((a, b) => b - a)
  return unique.length ? unique : [7, 3, 1]
}

function normalizeProvider(value: string): BackupStorageProvider {
  if (value === 'aliyun' || value === 'aliyun_oss' || value === 'oss') return 'aliyun_oss'
  if (value === 's3' || value === 'aws_s3' || value === 'compatible_s3') return 's3'
  return 'cloudflare_r2'
}

function normalizeInt(value: unknown, fallback: number): number {
  const numberValue = typeof value === 'number' ? value : Number.parseInt(String(value || ''), 10)
  if (!Number.isFinite(numberValue) || numberValue <= 0) return fallback
  return Math.floor(numberValue)
}

function formatBackupBytes(value: number): string {
  return value > 0 ? formatBytes(value) : '-'
}

function providerLabel(value: string): string {
  if (value === 'cloudflare_r2') return 'Cloudflare R2'
  if (value === 'aliyun_oss') return '阿里云 OSS'
  return 'S3 兼容'
}

function backupStatusLabel(value: string): string {
  if (value === 'completed') return '成功'
  if (value === 'failed') return '失败'
  if (value === 'running') return '运行中'
  if (value === 'pending') return '等待中'
  return value || '-'
}

function backupStatusClass(value: string): string {
  if (value === 'completed') return 'badge-success'
  if (value === 'failed') return 'badge-danger'
  if (value === 'running' || value === 'pending') return 'badge-warning'
  return 'badge-gray'
}

function restoreStatusLabel(value?: string): string {
  if (value === 'completed') return '已恢复'
  if (value === 'failed') return '恢复失败'
  if (value === 'running') return '恢复中'
  return '未恢复'
}

function restoreStatusClass(value?: string): string {
  if (value === 'completed') return 'badge-success'
  if (value === 'failed') return 'badge-danger'
  if (value === 'running') return 'badge-warning'
  return 'badge-gray'
}

function triggerLabel(value: string): string {
  if (value === 'scheduled') return '定时'
  if (value === 'manual') return '手动'
  return value || '-'
}

function renewalStateLabel(value?: string): string {
  if (value === 'active') return '正常'
  if (value === 'reminder_due') return '待提醒'
  if (value === 'due_today') return '今日到期'
  if (value === 'expired') return '已到期'
  return '未配置'
}

function renewalStateClass(value?: string): string {
  if (value === 'active') return 'badge-success'
  if (value === 'reminder_due' || value === 'due_today') return 'badge-warning'
  if (value === 'expired') return 'badge-danger'
  return 'badge-gray'
}

function renewalStateTextClass(value?: string): string {
  if (value === 'active') return 'text-emerald-700 dark:text-emerald-400'
  if (value === 'reminder_due' || value === 'due_today') return 'text-amber-600 dark:text-amber-400'
  if (value === 'expired') return 'text-rose-600 dark:text-rose-400'
  return 'text-gray-700 dark:text-dark-200'
}
</script>
