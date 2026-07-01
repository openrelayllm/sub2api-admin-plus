<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">成本对账</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            从供应商充值订单、兑换充值、用量消耗和余额快照生成上游成本台账。
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadCurrentTab">
            <Icon name="refresh" size="sm" />
            刷新
          </button>
          <button type="button" class="btn btn-primary" :disabled="backfilling" @click="startHistoryBackfill">
            <Icon name="play" size="sm" />
            {{ backfilling ? '回补中...' : '一键回补全部供应商' }}
          </button>
        </div>
      </section>

      <nav class="flex gap-2 overflow-x-auto border-b border-gray-200 dark:border-dark-700">
        <button
          v-for="tab in topTabs"
          :key="tab.value"
          type="button"
          class="whitespace-nowrap border-b-2 px-3 py-2 text-sm font-medium"
          :class="activeTopTab === tab.value ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-900 dark:text-dark-400 dark:hover:text-white'"
          @click="setTopTab(tab.value)"
        >
          {{ tab.label }}
        </button>
      </nav>

      <section class="grid gap-4 lg:grid-cols-[1.2fr_1fr_1fr_auto] lg:items-end">
        <label class="block">
          <span class="input-label">供应商</span>
          <select v-model.number="selectedSupplierId" class="input" @change="handleSupplierChange">
            <option :value="0">全部供应商</option>
            <option v-for="supplier in suppliers" :key="supplier.id" :value="supplier.id">{{ supplier.name }}</option>
          </select>
        </label>
        <label class="block">
          <span class="input-label">开始时间</span>
          <input v-model="syncForm.started_at" type="datetime-local" class="input" />
        </label>
        <label class="block">
          <span class="input-label">结束时间</span>
          <input v-model="syncForm.ended_at" type="datetime-local" class="input" />
        </label>
        <button type="button" class="btn btn-secondary" :disabled="syncing || !selectedSupplierId" @click="syncCosts">
          <Icon name="sync" size="sm" />
          {{ syncing ? '同步中' : '同步当前供应商' }}
        </button>
      </section>

      <section v-if="activeTopTab === 'overview'" class="card overflow-hidden">
        <div class="flex flex-col gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">总账统计</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">按币种汇总所有供应商最新成本快照，不跨币种折算。</p>
          </div>
          <div class="text-sm text-gray-500 dark:text-dark-400">
            {{ ledgerOverview?.generated_at ? `生成时间 ${formatDateTime(ledgerOverview.generated_at)}` : '暂无总账快照' }}
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[1260px] divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">币种</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">供应商/快照</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">充值总额</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">实际支付</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">充值订单</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">兑换充值</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">用量消耗</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">实际余额/快照</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">余额差异</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">最近采集</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="ledgerOverviewItems.length === 0">
                <td colspan="10" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无总账统计</td>
              </tr>
              <tr v-for="item in ledgerOverviewItems" :key="item.currency">
                <td class="px-4 py-4 text-sm font-semibold text-gray-900 dark:text-gray-100">{{ item.currency }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">
                  {{ item.supplier_count }} / {{ item.snapshot_count }}
                </td>
                <td class="px-4 py-4 text-right text-sm font-medium text-gray-900 dark:text-gray-100">{{ formatMoney(item.recharge_total_cents, item.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm font-semibold text-emerald-700 dark:text-emerald-300">{{ formatMoney(item.recharge_actual_payment_cents, item.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(item.completed_funding_amount_cents, item.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(item.entitlement_amount_cents, item.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(item.usage_cost_cents, item.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">
                  {{ item.actual_balance_cents === undefined || item.actual_balance_cents === null ? '-' : formatMoney(item.actual_balance_cents, item.currency) }}
                  <span class="ml-1 text-xs text-gray-400 dark:text-dark-500">({{ item.actual_balance_available_count }})</span>
                </td>
                <td class="px-4 py-4 text-right text-sm font-medium" :class="deltaClass(item.balance_delta_cents)">
                  {{ item.balance_delta_cents === undefined || item.balance_delta_cents === null ? '-' : formatMoney(item.balance_delta_cents, item.currency) }}
                </td>
                <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(item.latest_captured_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else-if="activeTopTab === 'suppliers'" class="card overflow-hidden">
        <div class="flex flex-col gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">供应商成本快照</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">点击供应商进入单供应商明细，避免首屏拉取所有明细。</p>
          </div>
          <div class="text-sm text-gray-500 dark:text-dark-400">{{ snapshots.length }} 条快照</div>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[1260px] divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">供应商</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">币种</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">充值总额</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">实际支付</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">充值订单</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">兑换充值</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">用量消耗</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">实际余额</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">差异</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">采集时间</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="snapshots.length === 0">
                <td colspan="10" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无成本快照</td>
              </tr>
              <tr
                v-for="snapshot in snapshots"
                :key="snapshot.id"
                class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-800"
                @click="selectSnapshot(snapshot.supplier_id)"
              >
                <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">{{ supplierName(snapshot.supplier_id) }}</td>
                <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">{{ snapshot.currency }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(supplierRechargeTotalCents(snapshot), snapshot.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm font-semibold text-emerald-700 dark:text-emerald-300">{{ formatMoney(snapshot.recharge_actual_payment_cents, snapshot.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(snapshot.completed_funding_amount_cents, snapshot.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(snapshot.entitlement_amount_cents, snapshot.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(supplierDisplayUsageCents(snapshot), snapshot.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ snapshot.actual_balance_cents === undefined || snapshot.actual_balance_cents === null ? '-' : formatMoney(snapshot.actual_balance_cents, snapshot.currency) }}</td>
                <td class="px-4 py-4 text-right text-sm" :class="deltaClass(supplierBalanceDeltaCents(snapshot))">
                  {{ supplierBalanceDeltaCents(snapshot) === null ? '-' : formatMoney(supplierBalanceDeltaCents(snapshot) || 0, snapshot.currency) }}
                </td>
                <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(snapshot.captured_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else-if="activeTopTab === 'detail'" class="space-y-6">
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-7">
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">充值总额</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatMoney(supplierRechargeTotalCents(currentSnapshot), currentCurrency) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">实际支付</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-700 dark:text-emerald-300">{{ formatMoney(currentSnapshot?.recharge_actual_payment_cents || 0, currentCurrency) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">充值订单</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatMoney(currentSnapshot?.completed_funding_amount_cents || 0, currentCurrency) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">兑换充值</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatMoney(currentSnapshot?.entitlement_amount_cents || 0, currentCurrency) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">用量消耗</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatMoney(supplierDisplayUsageCents(currentSnapshot), currentCurrency) }}</p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">实际余额</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ currentSnapshot?.actual_balance_cents === undefined || currentSnapshot?.actual_balance_cents === null ? '-' : formatMoney(currentSnapshot.actual_balance_cents, currentCurrency) }}
            </p>
          </div>
          <div class="card p-4">
            <p class="text-xs font-medium text-gray-500 dark:text-dark-400">余额差异</p>
            <p class="mt-2 text-2xl font-semibold" :class="balanceDeltaClass">
              {{ currentBalanceDelta === null ? '-' : formatMoney(currentBalanceDelta, currentCurrency) }}
            </p>
          </div>
        </div>

        <section class="card overflow-hidden">
          <div class="flex flex-col gap-4 border-b border-gray-100 px-5 py-4 dark:border-dark-700 lg:flex-row lg:items-center lg:justify-between">
            <div class="inline-flex w-fit rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-800">
              <button
                v-for="tab in detailTabs"
                :key="tab.value"
                type="button"
                class="rounded-md px-3 py-1.5 text-sm font-medium transition"
                :class="activeDetailTab === tab.value ? 'bg-white text-primary-600 shadow-sm dark:bg-dark-700 dark:text-primary-300' : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
                @click="setDetailTab(tab.value)"
              >
                {{ tab.label }}
              </button>
            </div>
            <div class="text-sm text-gray-500 dark:text-dark-400">{{ syncStatusLabel }}</div>
          </div>

          <div v-if="!selectedSupplierId" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-dark-400">
            请选择供应商查看明细。
          </div>
          <div v-else-if="activeDetailTab === 'summary'" class="px-5 py-8 text-sm text-gray-500 dark:text-dark-400">
            当前供应商：{{ supplierName(selectedSupplierId) }}，{{ currentSnapshot ? `最近采集 ${formatDateTime(currentSnapshot.captured_at)}` : '暂无快照' }}。
          </div>
          <div v-else-if="activeDetailTab === 'funding'" class="overflow-x-auto">
            <table class="w-full min-w-[1180px] divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">订单</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">支付方式</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">状态</th>
                  <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">额度</th>
                  <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">原始实付</th>
                  <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">充值倍率</th>
                  <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">实际支付</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">完成时间</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr v-if="funding.length === 0">
                  <td colspan="8" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无充值订单</td>
                </tr>
                <tr v-for="item in funding" :key="item.id">
                  <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">
                    <div class="font-mono text-xs">{{ item.external_id }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ item.out_trade_no || '-' }}</div>
                  </td>
                  <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">{{ item.payment_type || '-' }}</td>
                  <td class="px-4 py-4"><span class="badge badge-gray">{{ item.status }}</span></td>
                  <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(item.amount_cents, item.currency) }}</td>
                  <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(item.cash_amount_cents, item.currency) }}</td>
                  <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMultiplier(item.recharge_multiplier) }}</td>
                  <td class="px-4 py-4 text-right text-sm font-semibold text-emerald-700 dark:text-emerald-300">{{ formatMoney(item.actual_payment_cents, item.currency) }}</td>
                  <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(item.completed_at || item.paid_at || item.created_at_external) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else-if="activeDetailTab === 'entitlements'" class="overflow-x-auto">
            <table class="w-full min-w-[980px] divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">记录</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">来源</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">状态</th>
                  <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">权益内容</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">使用时间</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr v-if="entitlements.length === 0">
                  <td colspan="5" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无兑换记录</td>
                </tr>
                <tr v-for="item in entitlements" :key="item.id">
                  <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">
                    <div class="font-mono text-xs">{{ item.external_id }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">尾号 {{ item.code_last4 || '-' }}</div>
                  </td>
                  <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">
                    <div>{{ sourceFamilyLabel(item.source_family) }}</div>
                    <div class="mt-2">
                      <span class="badge" :class="entitlementBadgeClass(item.type)">{{ entitlementTypeLabel(item.type) }}</span>
                    </div>
                  </td>
                  <td class="px-4 py-4"><span class="badge badge-gray">{{ item.status }}</span></td>
                  <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ entitlementValueLabel(item) }}</td>
                  <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(item.used_at || item.created_at_external) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div v-else class="overflow-x-auto">
            <table class="w-full min-w-[980px] divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">类型</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">来源</th>
                  <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">金额</th>
                  <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">实际支付</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">发生时间</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr v-if="ledger.length === 0">
                  <td colspan="5" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无成本台账</td>
                </tr>
                <tr v-for="entry in ledger" :key="entry.id">
                  <td class="px-4 py-4"><span class="badge" :class="ledgerBadgeClass(entry.entry_type)">{{ ledgerTypeLabel(entry.entry_type) }}</span></td>
                  <td class="px-4 py-4 text-sm text-gray-900 dark:text-gray-100">
                    <div>{{ entry.source_type }}</div>
                    <div class="mt-1 font-mono text-xs text-gray-500 dark:text-dark-400">{{ entry.source_external_id || `#${entry.source_id}` }}</div>
                  </td>
                  <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ formatMoney(entry.amount_cents, entry.currency) }}</td>
                  <td class="px-4 py-4 text-right text-sm text-gray-900 dark:text-gray-100">{{ entry.actual_payment_cents ? formatMoney(entry.actual_payment_cents, entry.currency) : '-' }}</td>
                  <td class="px-4 py-4 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(entry.occurred_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </section>

      <section v-else class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div class="card overflow-hidden">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">历史回补运行</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">一键提交后由后台按供应商拆分 step 执行，页面只轮询 run 状态。</p>
              </div>
              <div class="flex flex-wrap gap-2">
                <button
                  v-if="activeBackfillRun"
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="cancellingRun || !runCancellable(activeBackfillRun.run.status)"
                  @click="cancelCurrentRun"
                >
                  取消队列/Run
                </button>
                <button
                  v-if="activeBackfillRun"
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="Boolean(deletingRunId) || !runDeletable(activeBackfillRun.run.status)"
                  @click="deleteRun(activeBackfillRun.run.id)"
                >
                  删除当前 Run
                </button>
                <button
                  type="button"
                  class="btn btn-danger btn-sm"
                  :disabled="clearingRuns || loadingRuns"
                  @click="clearFinishedCostRuns"
                >
                  {{ clearingRuns ? '清空中' : '清空已结束历史' }}
                </button>
              </div>
            </div>
          </div>
          <div v-if="!activeBackfillRun" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-dark-400">
            请选择一个历史回补 Run 查看明细。
          </div>
          <div v-else class="space-y-5 p-5">
            <div class="grid gap-3 md:grid-cols-5">
              <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
                <p class="text-xs text-gray-500 dark:text-dark-400">状态</p>
                <span class="badge mt-2" :class="runStatusClass(activeBackfillRun.run.status)">{{ runStatusLabel(activeBackfillRun.run.status) }}</span>
              </div>
              <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
                <p class="text-xs text-gray-500 dark:text-dark-400">供应商</p>
                <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ activeBackfillRun.run.supplier_count }}</p>
              </div>
              <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
                <p class="text-xs text-gray-500 dark:text-dark-400">成功</p>
                <p class="mt-2 text-sm font-medium text-emerald-600 dark:text-emerald-400">{{ activeBackfillRun.run.succeeded_steps }}</p>
              </div>
              <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
                <p class="text-xs text-gray-500 dark:text-dark-400">失败</p>
                <p class="mt-2 text-sm font-medium text-rose-600 dark:text-rose-400">{{ activeBackfillRun.run.failed_steps }}</p>
              </div>
              <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
                <p class="text-xs text-gray-500 dark:text-dark-400">总 Step</p>
                <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ activeBackfillRun.run.total_steps }}</p>
              </div>
            </div>

            <div class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-700">
              <table class="w-full min-w-[960px] divide-y divide-gray-200 dark:divide-dark-700">
                <thead class="bg-gray-50 dark:bg-dark-800">
                  <tr>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">Step</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">供应商</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">状态</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">结果</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">时间</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">错误/原因</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">操作</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
                  <tr v-if="activeBackfillRun.steps.length === 0">
                    <td colspan="7" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">暂无 step 明细</td>
                  </tr>
                  <tr v-for="step in activeBackfillRun.steps" :key="step.id">
                    <td class="px-4 py-3 font-mono text-xs text-gray-500 dark:text-dark-400">{{ step.id }}</td>
                    <td class="px-4 py-3 text-sm text-gray-900 dark:text-gray-100">{{ step.supplier_name || supplierName(step.supplier_id) }}</td>
                    <td class="px-4 py-3"><span class="badge" :class="runStatusClass(step.status)">{{ runStatusLabel(step.status) }}</span></td>
                    <td class="px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ stepResultLabel(step.result_snapshot, step.result_count) }}</td>
                    <td class="px-4 py-3 text-sm text-gray-500 dark:text-dark-400">
                      <div>{{ formatDateTime(step.started_at) }}</div>
                      <div v-if="step.finished_at" class="mt-1 text-xs text-gray-400 dark:text-dark-500">完成 {{ formatDateTime(step.finished_at) }}</div>
                    </td>
                    <td class="max-w-[260px] px-4 py-3 text-sm text-gray-500 dark:text-dark-400">
                      <button
                        v-if="step.reason"
                        type="button"
                        class="block max-w-full text-left hover:text-gray-900 dark:hover:text-gray-100"
                        @click="selectedReasonStep = step"
                      >
                        <span class="block truncate" :title="step.reason">{{ stepReasonSummary(step.reason) }}</span>
                        <span class="mt-1 block text-xs font-medium text-blue-700 dark:text-blue-300">查看详情</span>
                      </button>
                      <span v-else>-</span>
                    </td>
                    <td class="px-4 py-3">
                      <div class="flex flex-wrap gap-2">
                        <button type="button" class="btn btn-secondary btn-sm" :disabled="retryingStepId === step.id || !stepRetryable(step.status)" @click="retryStep(step.id)">重试</button>
                        <button type="button" class="btn btn-secondary btn-sm" :disabled="cancellingStepId === step.id || !stepCancellable(step.status)" @click="cancelStep(step.id)">取消</button>
                      </div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="flex flex-col gap-2 text-sm text-gray-500 dark:text-dark-400 sm:flex-row sm:items-center sm:justify-between">
              <div>Step 第 {{ stepPage }} 页，每页 {{ stepPageSize }} 条</div>
              <div class="flex gap-2">
                <button type="button" class="btn btn-secondary btn-sm" :disabled="loadingSteps || stepPage <= 1" @click="changeStepPage(stepPage - 1)">上一页</button>
                <button type="button" class="btn btn-secondary btn-sm" :disabled="loadingSteps || !stepHasNext" @click="changeStepPage(stepPage + 1)">下一页</button>
              </div>
            </div>

            <div class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-700">
              <table class="w-full min-w-[860px] divide-y divide-gray-200 dark:divide-dark-700">
                <thead class="bg-gray-50 dark:bg-dark-800">
                  <tr>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">Run</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">状态</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">供应商</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">Step</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">请求时间</th>
                    <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">操作</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
                  <tr v-if="costRuns.length === 0">
                    <td colspan="6" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400">暂无成本对账运行记录</td>
                  </tr>
                  <tr
                    v-for="run in costRuns"
                    :key="run.id"
                    class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-800"
                    :class="activeBackfillRun?.run.id === run.id ? 'bg-primary-50 dark:bg-primary-900/20' : ''"
                    @click="openCostRun(run.id)"
                  >
                    <td class="px-4 py-3 font-mono text-xs text-gray-500 dark:text-dark-400">{{ run.id }}</td>
                    <td class="px-4 py-3"><span class="badge" :class="runStatusClass(run.status)">{{ runStatusLabel(run.status) }}</span></td>
                    <td class="px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ run.supplier_count }}</td>
                    <td class="px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ run.succeeded_steps }}/{{ run.total_steps }} 成功，{{ run.failed_steps }} 失败</td>
                    <td class="px-4 py-3 text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(run.requested_at) }}</td>
                    <td class="px-4 py-3">
                      <button
                        type="button"
                        class="btn btn-secondary btn-sm"
                        :disabled="deletingRunId === run.id || !runDeletable(run.status)"
                        @click.stop="deleteRun(run.id)"
                      >
                        {{ runDeletable(run.status) ? '删除' : '先取消' }}
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div class="flex flex-col gap-2 text-sm text-gray-500 dark:text-dark-400 sm:flex-row sm:items-center sm:justify-between">
              <div>Run 第 {{ costRunPage }} 页，每页 {{ costRunPageSize }} 条</div>
              <div class="flex gap-2">
                <button type="button" class="btn btn-secondary btn-sm" :disabled="loadingRuns || costRunPage <= 1" @click="changeRunPage(costRunPage - 1)">上一页</button>
                <button type="button" class="btn btn-secondary btn-sm" :disabled="loadingRuns || !costRunsHasNext" @click="changeRunPage(costRunPage + 1)">下一页</button>
              </div>
            </div>
          </div>
        </div>

        <aside class="card p-5">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">回补范围</h2>
          <dl class="mt-4 space-y-3 text-sm">
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">目标</dt>
              <dd class="font-medium text-gray-900 dark:text-white">全部供应商</dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">开始</dt>
              <dd class="font-medium text-gray-900 dark:text-white">{{ formatDateTime(toRFC3339(syncForm.started_at)) }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">结束</dt>
              <dd class="font-medium text-gray-900 dark:text-white">{{ formatDateTime(toRFC3339(syncForm.ended_at)) }}</dd>
            </div>
            <div class="flex items-center justify-between gap-3">
              <dt class="text-gray-500 dark:text-dark-400">模式</dt>
              <dd class="font-medium text-gray-900 dark:text-white">后台分批</dd>
            </div>
          </dl>
          <button type="button" class="btn btn-primary mt-5 w-full" :disabled="backfilling" @click="startHistoryBackfill">
            <Icon name="play" size="sm" />
            {{ backfilling ? '已提交后台任务' : '提交历史回补' }}
          </button>
        </aside>
      </section>
    </div>

    <BaseDialog
      :show="Boolean(selectedReasonStep)"
      :title="selectedReasonStep ? `错误详情 - Step ${selectedReasonStep.id}` : '错误详情'"
      width="wide"
      @close="selectedReasonStep = null"
    >
      <div v-if="selectedReasonStep" class="space-y-5">
        <div class="grid gap-3 md:grid-cols-3">
          <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-xs text-gray-500 dark:text-dark-400">供应商</p>
            <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">
              {{ selectedReasonStep.supplier_name || supplierName(selectedReasonStep.supplier_id) }}
            </p>
          </div>
          <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-xs text-gray-500 dark:text-dark-400">状态</p>
            <span class="badge mt-2" :class="runStatusClass(selectedReasonStep.status)">{{ runStatusLabel(selectedReasonStep.status) }}</span>
          </div>
          <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-xs text-gray-500 dark:text-dark-400">Attempt</p>
            <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ selectedReasonStep.attempts }}/{{ selectedReasonStep.max_attempts }}</p>
          </div>
        </div>

        <dl class="grid gap-3 md:grid-cols-2">
          <div v-for="row in selectedReasonRows" :key="row.label" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <dt class="text-xs text-gray-500 dark:text-dark-400">{{ row.label }}</dt>
            <dd class="mt-2 break-words text-sm font-medium text-gray-900 dark:text-gray-100">{{ row.value || '-' }}</dd>
          </div>
        </dl>

        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <p class="text-xs text-gray-500 dark:text-dark-400">完整错误</p>
          <pre class="mt-2 max-h-72 overflow-auto whitespace-pre-wrap break-words rounded bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-800 dark:text-dark-200">{{ selectedRawReason }}</pre>
        </div>

        <div v-if="selectedReasonStep.result_snapshot || selectedReasonStep.request_snapshot" class="grid gap-3 md:grid-cols-2">
          <div v-if="selectedReasonStep.request_snapshot" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-xs text-gray-500 dark:text-dark-400">请求快照</p>
            <pre class="mt-2 max-h-56 overflow-auto whitespace-pre-wrap break-words rounded bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-800 dark:text-dark-200">{{ formatReasonSnapshot(selectedReasonStep.request_snapshot) }}</pre>
          </div>
          <div v-if="selectedReasonStep.result_snapshot" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-xs text-gray-500 dark:text-dark-400">结果快照</p>
            <pre class="mt-2 max-h-56 overflow-auto whitespace-pre-wrap break-words rounded bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-800 dark:text-dark-200">{{ formatReasonSnapshot(selectedReasonStep.result_snapshot) }}</pre>
          </div>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import {
  backfillSupplierCosts,
  cancelSchedulerRun,
  cancelSchedulerStep,
  deleteSchedulerRun,
  deleteSchedulerRuns,
  getSchedulerRunDetail,
  getSupplierCostLedgerOverview,
  getSupplierCostSummary,
  listSupplierCostLedger,
  listSupplierCostSnapshots,
  listSupplierEntitlementTransactions,
  listSupplierFundingTransactions,
  listSchedulerRuns,
  listSchedulerSteps,
  listSuppliers,
  retrySchedulerStep,
  type SchedulerRunDetail,
  type SchedulerRunSummary,
  type SchedulerStepRecord,
  type Supplier,
  type SupplierCostLedgerEntry,
  type SupplierCostLedgerOverview,
  type SupplierCostLedgerOverviewItem,
  type SupplierCostSnapshot,
  type SupplierEntitlementTransaction,
  type SupplierFundingTransaction
} from '@/api/admin/adminPlus'
import {
  supplierBalanceDeltaCents,
  supplierDisplayUsageCents,
  supplierRechargeTotalCents
} from './supplierCostPresentation'
import {
  actionLabel,
  codeFromReasonText,
  firstText,
  formatReasonSnapshot,
  metadataSummary,
  outcomeLabel,
  parseStepReason,
  plainStepReason,
  runStatusClass,
  runStatusLabel,
  runCancellable,
  runDeletable,
  stageLabel,
  stepCancellable,
  stepReasonSummary,
  stepRetryable,
  suggestionFromCode
} from '../scheduler/presentation'

type TopTab = 'overview' | 'suppliers' | 'detail' | 'backfill'
type DetailTab = 'summary' | 'funding' | 'entitlements' | 'ledger'

const appStore = useAppStore()
const loading = ref(false)
const syncing = ref(false)
const backfilling = ref(false)
const suppliersLoaded = ref(false)
const overviewLoaded = ref(false)
const snapshotsLoaded = ref(false)
const detailLoadedSupplierId = ref(0)
const suppliers = ref<Supplier[]>([])
const ledgerOverview = ref<SupplierCostLedgerOverview | null>(null)
const snapshots = ref<SupplierCostSnapshot[]>([])
const funding = ref<SupplierFundingTransaction[]>([])
const entitlements = ref<SupplierEntitlementTransaction[]>([])
const ledger = ref<SupplierCostLedgerEntry[]>([])
const selectedSupplierId = ref(0)
const activeTopTab = ref<TopTab>('overview')
const activeDetailTab = ref<DetailTab>('summary')
const activeBackfillRun = ref<SchedulerRunDetail | null>(null)
const costRuns = ref<SchedulerRunSummary[]>([])
const retryingStepId = ref<number | null>(null)
const cancellingStepId = ref<number | null>(null)
const cancellingRun = ref(false)
const deletingRunId = ref<string | null>(null)
const clearingRuns = ref(false)
const loadingRuns = ref(false)
const loadingSteps = ref(false)
const selectedReasonStep = ref<SchedulerStepRecord | null>(null)
let backfillRunTimer: number | undefined

const schedulerCostTaskType = 'supplier.costs.reconcile'
const legacySchedulerCostTaskType = 'reconcile_supplier_costs'
const costRunPageSize = 20
const stepPageSize = 100
const costRunPage = ref(1)
const costRunsHasNext = ref(false)
const stepPage = ref(1)
const stepHasNext = ref(false)
const defaultHistoryStartedAt = new Date(2020, 0, 1, 0, 0, 0)

const syncForm = reactive({
  started_at: toDateTimeLocal(defaultHistoryStartedAt),
  ended_at: toDateTimeLocal(new Date())
})

const topTabs: Array<{ value: TopTab; label: string }> = [
  { value: 'overview', label: '总账统计' },
  { value: 'suppliers', label: '供应商快照' },
  { value: 'detail', label: '供应商明细' },
  { value: 'backfill', label: '历史回补' }
]

const detailTabs: Array<{ value: DetailTab; label: string }> = [
  { value: 'summary', label: '成本摘要' },
  { value: 'funding', label: '充值订单' },
  { value: 'entitlements', label: '兑换记录' },
  { value: 'ledger', label: '成本台账' }
]

const currentSnapshot = computed(() => {
  return snapshots.value.find((item) => item.supplier_id === selectedSupplierId.value) || null
})
const currentCurrency = computed(() => currentSnapshot.value?.currency || 'USD')
const currentBalanceDelta = computed(() => supplierBalanceDeltaCents(currentSnapshot.value))
const balanceDeltaClass = computed(() => deltaClass(currentBalanceDelta.value))
const ledgerOverviewItems = computed<SupplierCostLedgerOverviewItem[]>(() => ledgerOverview.value?.items || [])
const syncStatusLabel = computed(() => {
  if (syncing.value) return '成本同步任务已提交到后台'
  if (activeBackfillRun.value) return runCaption(activeBackfillRun.value.run)
  if (!selectedSupplierId.value) return '选择供应商后同步成本'
  return '等待同步'
})
const selectedFailureReason = computed(() => parseStepReason(selectedReasonStep.value?.reason))
const selectedRawReason = computed(() => selectedReasonStep.value?.reason || '-')
const selectedReasonRows = computed(() => {
  const step = selectedReasonStep.value
  const reason = selectedFailureReason.value
  if (!step) return []
  const code = firstText(reason.login_code, reason.code, codeFromReasonText(step.reason || ''))
  return [
    { label: '阶段', value: stageLabel(reason.stage) },
    { label: '动作', value: actionLabel(reason.action || step.action) },
    { label: '结果', value: outcomeLabel(reason.outcome || step.status) },
    { label: '自动登录尝试', value: loginAttemptLabel(reason) },
    { label: '人工协助', value: manualRequiredLabel(reason, step.status) },
    { label: '错误码', value: code },
    { label: '错误信息', value: firstText(reason.login_message, reason.message, plainStepReason(step.reason || '')) },
    { label: '建议操作', value: reason.suggestion || suggestionFromCode(code) },
    { label: '上游诊断', value: metadataSummary(reason.metadata) },
    { label: '下次重试', value: formatDateTime(step.next_attempt_at) }
  ]
})

function loginAttemptLabel(reason: ReturnType<typeof parseStepReason>): string {
  if (reason.metadata?.login_attempted === 'true') return '已尝试一键登录'
  if (reason.metadata?.login_attempted === 'false') return '未尝试，缺少登录配置'
  if (reason.action === 'direct_login') return '已进入一键登录'
  return '未记录'
}

function manualRequiredLabel(reason: ReturnType<typeof parseStepReason>, status: string): string {
  if (reason.outcome === 'manual_required' || status === 'manual_required') return '需要人工协助'
  return '不需要，后台可重试'
}

function formatMoney(cents: number, currency: string): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: currency || 'USD',
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: 2
  }).format((cents || 0) / 100)
}

function formatNumber(value?: number | null): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(value || 0)
}

function formatMultiplier(value?: number | null): string {
  if (typeof value !== 'number') return '-'
  if (!Number.isFinite(value)) return '-'
  return `${value.toFixed(4).replace(/\.?0+$/, '')}x`
}

function formatDateTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString()
}

function toDateTimeLocal(value: Date): string {
  const offsetMs = value.getTimezoneOffset() * 60 * 1000
  return new Date(value.getTime() - offsetMs).toISOString().slice(0, 16)
}

function toRFC3339(value: string): string {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : date.toISOString()
}

function supplierName(id: number): string {
  return suppliers.value.find((supplier) => supplier.id === id)?.name || `#${id}`
}

function deltaClass(value?: number | null): string {
  if (value === null || value === undefined || value === 0) return 'text-emerald-600 dark:text-emerald-400'
  return 'text-rose-600 dark:text-rose-400'
}

function sourceFamilyLabel(value: string): string {
  return {
    payment_auto_redeem: '充值自动兑换',
    manual_redeem: '手工兑换'
  }[value] || value || '-'
}

function entitlementTypeLabel(value: string): string {
  return {
    balance: '余额',
    concurrency: '并发',
    subscription: '订阅'
  }[value] || value || '-'
}

function entitlementBadgeClass(value: string): string {
  if (value === 'balance') return 'badge-success'
  if (value === 'concurrency') return 'badge-warning'
  if (value === 'subscription') return 'badge-gray'
  return 'badge-gray'
}

function entitlementValueLabel(item: SupplierEntitlementTransaction): string {
  if (item.type === 'balance') return formatMoney(item.value_cents, item.currency)
  if (item.type === 'concurrency') return `+${formatNumber(item.raw_value)} 请求`
  if (item.type === 'subscription') return item.validity_days ? `${formatNumber(item.validity_days)} 天` : '订阅权益'
  if (item.raw_value !== undefined && item.raw_value !== null) return String(item.raw_value)
  return '-'
}

function ledgerTypeLabel(value: string): string {
  return {
    funding_credit: '充值入账',
    entitlement_credit: '兑换入账',
    refund_debit: '退款扣减',
    manual_adjustment: '手工调整',
    reversal: '冲正',
    usage_debit: '用量扣减'
  }[value] || value
}

function ledgerBadgeClass(value: string): string {
  if (value === 'refund_debit' || value === 'usage_debit') return 'badge-warning'
  if (value === 'manual_adjustment' || value === 'reversal') return 'badge-danger'
  return 'badge-success'
}

function isTerminalRunStatus(status: string): boolean {
  return ['succeeded', 'partial_succeeded', 'retryable_failed', 'manual_required', 'dead', 'cancelled', 'skipped'].includes(status)
}

function runCaption(run: SchedulerRunSummary): string {
  const prefix = `成本对账 run #${run.id} ${runStatusLabel(run.status)}`
  if (run.error_message) return `${prefix}：${run.error_message}`
  if (run.status === 'running') return `${prefix}，Worker 正在分批采集`
  if (run.status === 'queued') return `${prefix}，等待 Worker 执行`
  return prefix
}

function stepResultLabel(snapshot?: Record<string, unknown>, fallback = 0): string {
  if (!snapshot) return String(fallback || 0)
  const fundingCount = numberFromSnapshot(snapshot, 'funding_transactions')
  const entitlementCount = numberFromSnapshot(snapshot, 'entitlement_transactions')
  const usageCount = numberFromSnapshot(snapshot, 'usage_cost_lines')
  const ledgerCount = numberFromSnapshot(snapshot, 'ledger_entries')
  return `充值 ${fundingCount}，兑换 ${entitlementCount}，用量 ${usageCount}，台账 ${ledgerCount}`
}

function numberFromSnapshot(snapshot: Record<string, unknown>, key: string): number {
  const value = snapshot[key]
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

async function setTopTab(tab: TopTab) {
  activeTopTab.value = tab
  await loadCurrentTab()
}

async function setDetailTab(tab: DetailTab) {
  activeDetailTab.value = tab
  await loadDetailIfNeeded()
}

async function loadCurrentTab() {
  loading.value = true
  try {
    await loadSuppliersIfNeeded()
    if (activeTopTab.value === 'overview') {
      await loadLedgerOverview()
    } else if (activeTopTab.value === 'suppliers') {
      await loadSnapshots()
    } else if (activeTopTab.value === 'detail') {
      await ensureSelectedSupplier()
      await loadDetailIfNeeded(true)
    } else if (activeTopTab.value === 'backfill') {
      await loadCostRuns()
      if (activeBackfillRun.value) {
        await refreshBackfillRun(activeBackfillRun.value.run.id, { scheduleNext: true, notifyTerminal: false })
      } else if (costRuns.value[0]) {
        await openCostRun(costRuns.value[0].id, false)
      }
    }
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '加载成本对账失败')
  } finally {
    loading.value = false
  }
}

async function loadSuppliersIfNeeded() {
  if (suppliersLoaded.value) return
  const supplierResult = await listSuppliers({ limit: 200 })
  suppliers.value = supplierResult.items
  suppliersLoaded.value = true
}

async function loadLedgerOverview() {
  ledgerOverview.value = await getSupplierCostLedgerOverview()
  overviewLoaded.value = true
}

async function loadSnapshots() {
  const snapshotResult = await listSupplierCostSnapshots({ page: 1, page_size: 200 })
  snapshots.value = snapshotResult.items
  snapshotsLoaded.value = true
}

async function ensureSelectedSupplier() {
  if (selectedSupplierId.value) return
  selectedSupplierId.value = snapshots.value[0]?.supplier_id || suppliers.value[0]?.id || 0
}

async function loadDetailIfNeeded(force = false) {
  if (!selectedSupplierId.value) return
  if (!force && detailLoadedSupplierId.value === selectedSupplierId.value) return
  const [summaryResult, fundingResult, entitlementResult, ledgerResult] = await Promise.all([
    getSupplierCostSummary(selectedSupplierId.value),
    listSupplierFundingTransactions(selectedSupplierId.value, { page: 1, page_size: 100 }),
    listSupplierEntitlementTransactions(selectedSupplierId.value, { page: 1, page_size: 100 }),
    listSupplierCostLedger(selectedSupplierId.value, { page: 1, page_size: 100 })
  ])
  const others = snapshots.value.filter((item) => item.supplier_id !== selectedSupplierId.value)
  snapshots.value = [...summaryResult.items, ...others]
  funding.value = fundingResult.items
  entitlements.value = entitlementResult.items
  ledger.value = ledgerResult.items
  snapshotsLoaded.value = true
  detailLoadedSupplierId.value = selectedSupplierId.value
}

function handleSupplierChange() {
  detailLoadedSupplierId.value = 0
  if (activeTopTab.value === 'detail' && selectedSupplierId.value) {
    void loadCurrentTab()
  }
}

function selectSnapshot(supplierID: number) {
  selectedSupplierId.value = supplierID
  detailLoadedSupplierId.value = 0
  activeTopTab.value = 'detail'
  void loadCurrentTab()
}

async function syncCosts() {
  if (!selectedSupplierId.value) {
    appStore.showError('请选择供应商')
    return
  }
  stopBackfillPolling()
  syncing.value = true
  activeTopTab.value = 'backfill'
  try {
    const run = await backfillSupplierCosts({
      ...syncPayload(),
      supplier_id: selectedSupplierId.value
    })
    appStore.showSuccess(`当前供应商成本同步已提交 #${run.id}`)
    costRunPage.value = 1
    await loadCostRunsPage(1)
    await watchBackfillRun(run.id)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '同步成本失败')
    syncing.value = false
  }
}

async function startHistoryBackfill() {
  stopBackfillPolling()
  backfilling.value = true
  activeTopTab.value = 'backfill'
  try {
    const run = await backfillSupplierCosts({
      ...syncPayload()
    })
    appStore.showSuccess(`历史回补已提交 #${run.id}`)
    costRunPage.value = 1
    await loadCostRunsPage(1)
    await watchBackfillRun(run.id)
  } catch (error) {
    backfilling.value = false
    appStore.showError((error as { message?: string }).message || '提交历史回补失败')
  }
}

function syncPayload() {
  return {
    started_at: toRFC3339(syncForm.started_at),
    ended_at: toRFC3339(syncForm.ended_at),
    include_funding_transactions: true,
    include_entitlement_transactions: true,
    include_usage_cost_lines: true,
    include_balance_snapshot: true
  }
}

async function watchBackfillRun(runID: string) {
  stopBackfillPolling()
  stepPage.value = 1
  await refreshBackfillRun(runID, { scheduleNext: true, notifyTerminal: true })
}

async function refreshBackfillRun(runID: string, options: { scheduleNext?: boolean; notifyTerminal?: boolean } = {}) {
  const scheduleNext = options.scheduleNext !== false
  const notifyTerminal = options.notifyTerminal === true
  try {
    const detail = await getSchedulerRunDetail(runID)
    activeBackfillRun.value = detail
    await loadBackfillSteps(runID, stepPage.value)
    if (isTerminalRunStatus(detail.run.status)) {
      backfilling.value = false
      syncing.value = false
      await loadCostRuns()
      if (snapshotsLoaded.value) await loadSnapshots()
      if (overviewLoaded.value) await loadLedgerOverview()
      if (!notifyTerminal) {
        return
      }
      if (detail.run.status === 'succeeded' || detail.run.status === 'partial_succeeded') {
        appStore.showSuccess('成本对账后台任务完成')
      } else if (detail.run.error_message) {
        appStore.showError(detail.run.error_message)
      }
      return
    }
    if (!scheduleNext) return
    backfillRunTimer = window.setTimeout(() => {
      void refreshBackfillRun(runID, { scheduleNext: true, notifyTerminal: true })
    }, 2500)
  } catch (error) {
    backfilling.value = false
    syncing.value = false
    appStore.showError((error as { message?: string }).message || '读取历史回补状态失败')
  }
}

async function loadCostRuns() {
  await loadCostRunsPage(costRunPage.value)
}

async function loadCostRunsPage(page: number) {
  const normalizedPage = Math.max(1, page)
  loadingRuns.value = true
  try {
    const runs = await listSchedulerRuns({
      limit: costRunPageSize + 1,
      offset: (normalizedPage - 1) * costRunPageSize,
      task_type: schedulerCostTaskType
    })
    costRunsHasNext.value = runs.length > costRunPageSize
    costRuns.value = runs.slice(0, costRunPageSize)
    costRunPage.value = normalizedPage
  } finally {
    loadingRuns.value = false
  }
}

async function changeRunPage(page: number) {
  await loadCostRunsPage(page)
}

async function openCostRun(runID: string, showError = true) {
  try {
    stopBackfillPolling()
    stepPage.value = 1
    const detail = await getSchedulerRunDetail(runID)
    activeBackfillRun.value = detail
    await loadBackfillSteps(runID, 1)
    if (!isTerminalRunStatus(detail.run.status)) {
      backfillRunTimer = window.setTimeout(() => {
        void refreshBackfillRun(runID, { scheduleNext: true, notifyTerminal: false })
      }, 2500)
    }
  } catch (error) {
    if (showError) appStore.showError((error as { message?: string }).message || '读取运行详情失败')
  }
}

async function loadBackfillSteps(runID: string, page: number) {
  const normalizedPage = Math.max(1, page)
  loadingSteps.value = true
  try {
    const rows = await listSchedulerSteps({
      run_id: runID,
      limit: stepPageSize + 1,
      offset: (normalizedPage - 1) * stepPageSize
    })
    stepHasNext.value = rows.length > stepPageSize
    stepPage.value = normalizedPage
    if (activeBackfillRun.value?.run.id === runID) {
      activeBackfillRun.value = {
        ...activeBackfillRun.value,
        steps: rows.slice(0, stepPageSize)
      }
    }
  } finally {
    loadingSteps.value = false
  }
}

async function changeStepPage(page: number) {
  const runID = activeBackfillRun.value?.run.id
  if (!runID) return
  await loadBackfillSteps(runID, page)
}

async function retryStep(stepID: number) {
  retryingStepId.value = stepID
  try {
    await retrySchedulerStep(stepID)
    if (activeBackfillRun.value) await refreshBackfillRun(activeBackfillRun.value.run.id, { scheduleNext: true, notifyTerminal: false })
    await loadCostRuns()
    appStore.showSuccess('Step 已重新排队')
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '重试 step 失败')
  } finally {
    retryingStepId.value = null
  }
}

async function cancelStep(stepID: number) {
  cancellingStepId.value = stepID
  try {
    await cancelSchedulerStep(stepID)
    if (activeBackfillRun.value) await refreshBackfillRun(activeBackfillRun.value.run.id, { scheduleNext: true, notifyTerminal: false })
    await loadCostRuns()
    appStore.showSuccess('Step 已取消')
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '取消 step 失败')
  } finally {
    cancellingStepId.value = null
  }
}

async function cancelCurrentRun() {
  const runID = activeBackfillRun.value?.run.id
  if (!runID || cancellingRun.value) return
  cancellingRun.value = true
  try {
    stopBackfillPolling()
    await cancelSchedulerRun(runID)
    await refreshBackfillRun(runID, { scheduleNext: false, notifyTerminal: false })
    await loadCostRuns()
    backfilling.value = false
    syncing.value = false
    appStore.showSuccess('Run 已取消')
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '取消 run 失败')
  } finally {
    cancellingRun.value = false
  }
}

async function deleteRun(runID: string) {
  if (!runID || deletingRunId.value) return
  const confirmed = window.confirm(`确认删除 Run ${runID} 的历史记录？只会删除 scheduler run/step/attempt，不会删除成本台账。`)
  if (!confirmed) return
  deletingRunId.value = runID
  try {
    if (activeBackfillRun.value?.run.id === runID) {
      stopBackfillPolling()
    }
    const result = await deleteSchedulerRun(runID)
    if (activeBackfillRun.value?.run.id === runID) {
      activeBackfillRun.value = null
      stepPage.value = 1
      stepHasNext.value = false
    }
    await loadCostRuns()
    appStore.showSuccess(`已删除 Run ${runID}：${result.deleted_steps} 个 step`)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '删除 run 失败')
  } finally {
    deletingRunId.value = null
  }
}

async function clearFinishedCostRuns() {
  if (clearingRuns.value) return
  const confirmed = window.confirm('确认清空所有已结束的成本回补历史？queued/running 的任务不会删除，成本台账不会删除。')
  if (!confirmed) return
  clearingRuns.value = true
  try {
    stopBackfillPolling()
    const result = await deleteSchedulerRuns({ task_type: schedulerCostTaskType })
    const legacyResult = await deleteSchedulerRuns({ task_type: legacySchedulerCostTaskType })
    activeBackfillRun.value = null
    costRunPage.value = 1
    stepPage.value = 1
    stepHasNext.value = false
    await loadCostRunsPage(1)
    appStore.showSuccess(`已清空 ${result.deleted_runs + legacyResult.deleted_runs} 个 Run、${result.deleted_steps + legacyResult.deleted_steps} 个 Step`)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || '清空历史失败')
  } finally {
    clearingRuns.value = false
  }
}

function stopBackfillPolling() {
  if (!backfillRunTimer) return
  window.clearTimeout(backfillRunTimer)
  backfillRunTimer = undefined
}

onMounted(loadCurrentTab)
onBeforeUnmount(() => {
  stopBackfillPolling()
})
</script>
